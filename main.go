package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

//
const MAJOR = "1.0"

var VERSION = "1"

//
type application struct {
	List struct {
		Separator string
	}

	AWS struct {
		Timeout            int
		Host, Path, Prefix string
	}

	MissingKey string

	Output struct {
		Verbose, Debugging, Silent bool
		Mock                       bool
	}
}

//
func (config application) String() string {
	if buf, err := yaml.Marshal(config); err != nil {
		return fmt.Sprintf("%#v", config)
	} else {
		return string(buf)
	}
}

//
var settings application

//
func main() {
	app := cli.NewApp()
	app.Name = "counselor"
	app.Usage = "Runs a command within AWS and exposes all the AWS metadata to the command."
	app.ArgsUsage = "-- command parameters..."
	app.Version = MAJOR + "." + VERSION
	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		{
			Name:      "run",
			Aliases:   []string{"r"},
			Usage:     "Run the command after template-processing the parameters and environment.",
			ArgsUsage: "[options] -- command parameters...",
			Action:    run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "metadata-host",
					Value:       "169.254.169.254",
					EnvVar:      "METADATA_HOST",
					Destination: &settings.AWS.Host,
				},
				cli.StringFlag{
					Name:        "metadata-path",
					Value:       "latest/meta-data/",
					EnvVar:      "METADATA_PATH",
					Destination: &settings.AWS.Path,
				},
				cli.IntFlag{
					Name:        "metadata-timeout",
					Usage:       "Seconds to wait for GET response.",
					Value:       5,
					EnvVar:      "METADATA_TIMEOUT",
					Destination: &settings.AWS.Timeout,
				},
				cli.StringFlag{
					Name:        "prefix",
					Usage:       "The environment variable prefix for metadata keys.",
					Value:       "AWS_METADATA_",
					Destination: &settings.AWS.Prefix,
				},
				cli.StringFlag{
					Name:        "separator",
					Usage:       "The string that replaces carriage returns in metadata values.",
					Value:       " ",
					Destination: &settings.List.Separator,
				},
				cli.StringFlag{
					Name:        "missing",
					Usage:       "The value used when referencing a nonexistant template key.",
					Value:       "error",
					Destination: &settings.MissingKey,
				},
				cli.BoolFlag{
					Name:        "silent, S",
					Usage:       "Disable all output, including errors.",
					Destination: &settings.Output.Silent,
				},
				cli.BoolFlag{
					Name:        "verbose, V",
					Usage:       "Report on the settings, metadata, and the environment.",
					Destination: &settings.Output.Verbose,
				},
				cli.BoolFlag{
					Name:        "debug, debugging, D",
					Usage:       "Extensive output.",
					Destination: &settings.Output.Debugging,
				},
				cli.BoolFlag{
					Name:         "mock, M",
					Usage:        "Mock output.",
					EnvVar:       "COUNSELOR_MOCK",
					Destination:  &settings.Output.Mock,
				},
			},
			Before: func(ctx *cli.Context) error {
				if settings.Output.Silent {
					settings.Output.Debugging = false
					settings.Output.Verbose = false
				}

				if settings.Output.Debugging {
					settings.Output.Verbose = true
				}

				if settings.Output.Verbose {
					log.Printf("settings:\n%s", settings)
				}

				args := ctx.Args()
				if len(args) == 0 {
					return errors.New("No command provided.")
				}

				switch settings.MissingKey {
				case "zero", "error", "default", "invalid":
				default:
					fmt.Fprintf(os.Stderr, "ERROR: Resetting invalid missingkey: %+v", settings.MissingKey)
					settings.MissingKey = "default"
				}
				settings.MissingKey = fmt.Sprintf("missingkey=%s", settings.MissingKey)

				return nil
			},
		},
		{
			Name:      "test",
			Aliases:   []string{"r"},
			Usage:     "Test command.",
			ArgsUsage: "-- anything ...",
			Action:    test,
		},
	}

	app.Run(os.Args)
}

//
func test(ctx *cli.Context) error {
	fmt.Print("args:")
	for _, arg := range ctx.Args() {
		fmt.Printf("\n\t%s", arg)
	}
	fmt.Print("\nenv:")
	var env sort.StringSlice = os.Environ()
	env.Sort()
	for _, env := range env {
		fmt.Printf("\n\t%s", env)
	}
	fmt.Println()
	return nil
}

//
func run(ctx *cli.Context) error {
	var data metadataMap
	if settings.Output.Mock {
		data = metadataMap{
			"CounselorMock":"true",
		}
	} else {
		baseUrl := fmt.Sprintf("http://%s/%s", settings.AWS.Host, settings.AWS.Path)
		d, err := listed(baseUrl)
		if err != nil {
			return err
		}
		data = d
	}

	// Prefix the data prior to adding anything else.
	prefixed := makeEnv(settings.AWS.Prefix, data)

	var env = sort.StringSlice{
		fmt.Sprintf("COUNSELOR_STARTED=%d", time.Now().UTC().Unix()),
		fmt.Sprintf("COUNSELOR_VERSION=%s.%s", MAJOR, VERSION),
	}

	// Check for containerization

	func() {
		// cat /proc/self/cgroup | awk -F'/' '{print $3}'
		f, err := os.Open("/proc/self/cgroup")
		if err != nil {
			return
		}
		defer f.Close()
		r := bufio.NewReader(f)
		first, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if parts := strings.Split(strings.TrimSpace(first), "/"); len(parts) < 3 || parts[1] != "docker" {
			if settings.Output.Verbose {
				log.Printf("Not a properly formatted /proc/self/cgroup file: %+v", parts)
			}
		} else {
			env = append(env, fmt.Sprintf("COUNSELOR_CONTAINERID=%s", parts[2]))
		}
	}()

	env = append(env, os.Environ()...)
	env = append(env, prefixed...)
	env.Sort()

	{
		v := make(map[string]string)
		for _, item := range env {
			splits := strings.Split(item, "=")
			v[splits[0]] = join("=", splits[1:])
		}
		data["env"] = v
	}

	env = render(env, data)

	if settings.Output.Verbose {
		log.Printf("variables:\n%s", data)
	}

	args := ctx.Args()

	if settings.Output.Debugging {
		log.Printf("raw command: %+v", args)
	}

	final := render(args, data)

	binary, err := exec.LookPath(final[0])
	if err != nil {
		return err
	}

	if !settings.Output.Silent {
		log.Printf("executing: %+v", final)
	}
	return syscall.Exec(binary, final, env)
}

//
type metadataMap map[string]interface{}

//
func (md metadataMap) String() string {
	if buf, err := yaml.Marshal(md); err != nil {
		return fmt.Sprintf("%#v", md)
	} else {
		return string(buf)
	}
}

//
var safe_re = regexp.MustCompile(`[^A-Z0-9_]`)

//
func render(args []string, data metadataMap) []string {
	for i, arg := range args {
		if !strings.Contains(arg, "{{") {
			continue
		}
		tmpl, err := template.New(arg).
			Option(settings.MissingKey).
			Funcs(funcs).
			Parse(arg)
		if err != nil {
			if settings.Output.Debugging {
				log.Print(err)
			}
			continue
		}
		var narg bytes.Buffer
		err = tmpl.Execute(&narg, data)
		if err != nil {
			if settings.Output.Debugging {
				log.Print(err)
			}
			continue
		}
		args[i] = narg.String()
	}
	return args
}

//
func makeEnv(prefix string, metadata metadataMap) []string {
	out := make([]string, 0)
	for n, v := range metadata {
		safe := safe_re.ReplaceAllString(strings.ToUpper(n), "_")
		switch s := v.(type) {
		case string:
			s = fmt.Sprintf("%s%s=%s", prefix, safe, s)
			out = append(out, s)
		case metadataMap:
			out = append(out, makeEnv(fmt.Sprintf("%s%s_", prefix, safe), s)...)
		}
	}
	return out
}

//
func listed(url string) (metadataMap, error) {
	data, err := get(url)
	if err != nil {
		return nil, err
	}
	return metadata(url, strings.Split(string(data), "\n"))
}

//
func camel(name string) string {
	parts := strings.Split(name, "-")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

//
func metadata(url string, list []string) (metadataMap, error) {

	data := metadataMap{}
	for _, m := range list {
		n := string(m)
		c := camel(n)
		if !strings.HasSuffix(n, "/") {
			body, err := get(url + n)
			if err != nil {
				if !settings.Output.Silent {
					log.Printf("ERROR: %s: %s", n, err)
				}
				continue
			}
			v := strings.Replace(string(body), "\n", settings.List.Separator, -1)
			data[c] = v
			if settings.Output.Debugging {
				log.Println(n, v)
			}
		} else {
			meta, err := listed(url + n)
			if err != nil {
				if !settings.Output.Silent {
					log.Printf("ERROR: %s: %s", n, err)
				}
				continue
			}
			data[c[:len(c) - 1]] = meta
		}
	}

	return data, nil
}

//
func get(url string) ([]byte, error) {
	timeout := time.Duration(time.Duration(settings.AWS.Timeout) * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	if settings.Output.Debugging {
		log.Printf("GET %s", url)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		if !settings.Output.Silent {
			log.Printf("ERROR: %d : %s", resp.StatusCode, resp.Header)
		}
		return nil, errors.New(resp.Status)
	}

	return body, nil
}
