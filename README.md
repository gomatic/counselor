# counselor

[![Build Status](https://travis-ci.org/gomatic/counselor.svg?branch=master)](https://travis-ci.org/gomatic/counselor)

Runs a comand after template-processing the parameters and environment with AWS
instance metadata provide as template variables.

    go get github.com/gomatic/counselor

## What it does

1. obtains the metdata from `169.254.169.254/latest/meta-data/`.
1. renders template `{{.variables}}` in command-line parameters and environment variables.
1. `exec` the provided command.

## e.g.

On an AWS instance

    go get github.com/gomatic/counselor

Test using `/bin/echo`:

    counselor run --silent -- /bin/echo {{.LocalIpv4}}

Show verbose output:

    counselor run --verbose -- /bin/echo {{.LocalIpv4}}

Test using the provided debugger:

    counselor run --silent -- counselor test -- {{.LocalIpv4}}

Notice that `counselor` processes environment variables too:

    AZ={{.Placement.AvailabilityZone}} counselor run --silent -- counselor test -- {{.LocalIpv4}} | grep AZ


# AWS Metadata

The metadata is a map-tree of strings similar to this (YMMV):

    AmiId:
    AmiLaunchIndex:
    AmiManifestPath:
    BlockDeviceMapping:
      Ami:
      Root:
    Hostname:
    InstanceAction:
    InstanceId:
    InstanceType:
    KernelId:
    LocalHostname:
    LocalIpv4:
    Mac:
    Metrics:
      Vhostmd:
    Network:
      Interfaces:
        Macs:
          XX:XX:XX:XX:XX:XX:
            DeviceNumber:
            InterfaceId:
            Ipv4Associations:
              XX.XX.XX.XX:
            LocalHostname:
            LocalIpv4s:
            Mac:
            OwnerId:
            PublicHostname:
            PublicIpv4s:
            SecurityGroupIds:
            SecurityGroups:
            SubnetId:
            SubnetIpv4CidrBlock:
            VpcId:
            VpcIpv4CidrBlock:
            VpcIpv4CidrBlocks:
    Placement:
      AvailabilityZone:
    Profile:
    PublicHostname:
    PublicIpv4:
    PublicKeys:
    ReservationId:
    SecurityGroups:
    Services:
      Domain:
      Partition:
