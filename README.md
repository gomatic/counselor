# counselor

[![Build Status](https://travis-ci.org/gomatic/counselor.svg?branch=master)](https://travis-ci.org/gomatic/counselor)

Runs a command after template-processing the parameters and environment using AWS
instance metadata provided as template values.

## What it does

On an AWS instance:

1. obtains the metadata from `instance-data/latest/meta-data/`.
1. renders template `{{.variables}}` in command-line parameters and environment variables.
1. `exec` the provided command.

## Install

On an AWS instance:

    go get github.com/gomatic/counselor

## Examples

On an AWS instance:

Test using `/bin/echo` to print the instance's private IPv4 address and AZ:

    counselor run --silent -- /bin/echo {{.LocalIpv4}} {{.Placement.AvailabilityZone}}

Everything after the bare `--` is template processed and then executed.

To help identify what template variables are available (variables are case-sensitive), use verbose mode:

    counselor run --verbose -- /bin/echo {{.LocalIpv4}} {{.Placement.AvailabilityZone}}

### Interrogating the variables and environment

There is a test mode:

    counselor test -- {{.LocalIpv4}} {{.Placement.AvailabilityZone}}

Running by itself like that doesn't show any of the `AWS_*` environment and its arguments remain the literals passed. All the tester does is dump it's arguments and the environment variables defined for its process.

But we can use that tester to see what `counselor run` is actually doing. Compare the above test with this test:

    counselor run --silent -- counselor test -- {{.LocalIpv4}} {{.Placement.AvailabilityZone}}

You'll see (scrolling back through the output) that it has added lots of `AWS_*` variables to the environment and it has template processed the command-line arguments. That is, the `counselor test` that `counselor run` executes, actually only sees the local IPv4 address and AZ on the command line, not the template variables.

### Advanced Examples

Notice that `counselor` processes environment variables too:

    MYDATA="{{.LocalIpv4}},{{.Placement.AvailabilityZone}}" counselor run --silent -- counselor test | grep MYDATA

That command is:

1. Adding `MYDATA="{{.LocalIpv4}},{{.Placement.AvailabilityZone}}"` to `counselor run`'s environment.
1. `counselor run` template processes the environment.
1. `counselor run` runs `counselor test` which dumps its environment.
1. `grep` filters just the `MYDATA` value that `counselor test` sees as a string that contains the local IPv4 and AZ.

#### Using Functions

Functions add quite a lot of capability. 

One useful function is simple IP math. The following will increment the IP's 3rd group (zero-based, left-to-right) between `10 <= X < 15`.
This might be useful to, for example, auto-configure a cluster to communicate with other well-know host IPs but for which
it's not ideal to store the IP in the command-line.

    counselor run --silent -- counselor test -- '{{.LocalIpv4 | ip4_next 3 10 5}}'

Imagine your cluster is `192.168.1.10` through `192.168.1.14`. With the above template, each node can be configured to
communicate with the next node in the cluster, as a ring.

So `{{"192.168.1.14" | ip4_next 3 10 5}}` means:
- take IP group `3` which is `14`
- with the lowest allowed value for group `3` being `10`,
- compute the next IP for a `5` node cluster which is `192.168.1.10` (wraps the ring).

All of the functions are defined in [funcmap.go](https://github.com/gomatic/funcmap/blob/master/funcs.go) with [examples](https://github.com/gomatic/renderizer/blob/master/test/functions.txt) in the [renderizer repository](https://github.com/gomatic/renderizer).

_Disclaimer: These functions might seem convoluted but one of the goals is to be able to compute IP addresses without adding any quoting
in the template. Quoting templates within a command line that itself requires quoting can be confusing and complex._

# AWS Metadata Environment Variables Syntax

The metadata environment variables are generated based on the hierarchy of the variables, similar to the following (YMMV).
Counselor iterates the instance data at the time of execution so if instance data is added in the future, counselor will reflect it automatically.

    AmiId:                       AWS_METADATA_AMIID
    AmiLaunchIndex:              AWS_METADATA_AMILAUNCHINDEX
    AmiManifestPath:             AWS_METADATA_AMIMANIFESTPATH
    BlockDeviceMapping:
      Ami:                       AWS_METADATA_BLOCKDEVICEMAPPING_AMI
      Root:                      AWS_METADATA_BLOCKDEVICEMAPPING_ROOT
    Hostname:                    AWS_METADATA_HOSTNAME
    InstanceAction:              AWS_METADATA_INSTANCEACTION
    InstanceId:                  AWS_METADATA_INSTANCEID
    InstanceType:                AWS_METADATA_INSTANCETYPE
    KernelId:                    AWS_METADATA_KERNELID
    LocalHostname:               AWS_METADATA_LOCALHOSTNAME
    LocalIpv4:                   AWS_METADATA_LOCALIPV4
    Mac:                         AWS_METADATA_MAC
    Metrics:
      Vhostmd:                   AWS_METADATA_METRICS_VHOSTMD
    Network:
      Interfaces:
        Macs:
          XX:XX:XX:XX:XX:XX:
            DeviceNumber:        AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_DEVICENUMBER
            InterfaceId:         AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_INTERFACEID
            Ipv4Associations:
              XX.XX.XX.XX:       AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_IPV4ASSOCIATIONS_XX_XX_XX_XX
            LocalHostname:       AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_LOCALHOSTNAME
            LocalIpv4s:          AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_LOCALIPV4S
            Mac:                 AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_MAC
            OwnerId:             AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_OWNERID
            PublicHostname:      AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_PUBLICHOSTNAME
            PublicIpv4s:         AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_PUBLICIPV4S
            SecurityGroupIds:    AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_SECURITYGROUPIDS
            SecurityGroups:      AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_SECURITYGROUPS
            SubnetId:            AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_SUBNETID
            SubnetIpv4CidrBlock: AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_SUBNETIPV4CIDRBLOCK
            VpcId:               AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_VPCID
            VpcIpv4CidrBlock:    AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_VPCIPV4CIDRBLOCK
            VpcIpv4CidrBlocks:   AWS_METADATA_NETWORK_INTERFACES_MACS_XX_XX_XX_XX_XX_XX_VPCIPV4CIDRBLOCKS
    Placement:
      AvailabilityZone:          AWS_METADATA_PLACEMENT_AVAILABILITYZONE
    Profile:                     AWS_METADATA_PROFILE
    PublicHostname:              AWS_METADATA_PUBLICHOSTNAME
    PublicIpv4:                  AWS_METADATA_PUBLICIPV4
    PublicKeys:
    ReservationId:               AWS_METADATA_RESERVATIONID
    SecurityGroups:              AWS_METADATA_SECURITYGROUPS
    Services:
      Domain:                    AWS_METADATA_SERVICES_DOMAIN
      Partition:                 AWS_METADATA_SERVICES_PARTITION
