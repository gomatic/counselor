package main

import (
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

	Output struct {
		Verbose, Debugging, Silent bool
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
	baseUrl := fmt.Sprintf("http://%s/%s", settings.AWS.Host, settings.AWS.Path)
	data, err := listed(baseUrl)
	if err != nil {
		return err
	}
	if settings.Output.Verbose {
		log.Printf("metadata:\n%s", data)
	}

	args := ctx.Args()

	if settings.Output.Debugging {
		log.Printf("raw command: %+v", args)
	}

	args = render(args, data)

	binary, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}
	var env sort.StringSlice
	env = append(env, os.Environ()...)
	env = append(env, makeEnv(settings.AWS.Prefix, data)...)
	env.Sort()

	env = render(env, data)
	if settings.Output.Verbose {
		if buf, err := yaml.Marshal(env); err != nil {
			log.Printf("environment: %s", env)
		} else {
			log.Printf("environment:\n%s", buf)
		}
	}
	if !settings.Output.Silent {
		log.Printf("executing: %+v", args)
	}
	return syscall.Exec(binary, args, env)
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
		tmpl, err := template.New(arg).Parse(arg)
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
