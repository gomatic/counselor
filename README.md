# counselor

[![Build Status](https://travis-ci.org/gomatic/counselor.svg?branch=master)](https://travis-ci.org/gomatic/counselor)

Runs a command after template-processing the parameters and environment with AWS
instance metadata provide as template variables.

    go get github.com/gomatic/counselor

## What it does

1. obtains the metadata from `169.254.169.254/latest/meta-data/`.
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

Though that's redundant since `Placement.AvailabilityZone` is accessible in the environment as `AWS_METADATA_PLACEMENT_AVAILABILITYZONE`.
So environment variable templates will likely be most useful to apply functions.

And there is simple IP math. The following will increment the IP's 3rd group (zero-based, left-to-right) between `10 <= X < 15`.
This might be useful to, for example, auto-configure a cluster to communicate with other well-know host IPs but for which
it's not ideal to store the IP in the command-line.

    counselor run --silent -- counselor test -- {{.LocalIpv4 | ip4_next 3 10 5}}

Imagine your cluster is `192.168.1.10` through `192.168.1.14`. With the above template, each node can be configured to
communicate with the next node in the cluster, as a ring.

So `{{"192.168.1.14" | ip4_next 3 10 5}}` means:
- take IP group `3` which is `14`
- with the lowest value for group `3` being `10`,
- compute the next IP for a `5` node cluster which is `192.168.1.10` (wraps the ring).

_Disclaimer: This might convoluted but one of the goals is to be able to compute IP addresses without adding any quoting
in the template._

# AWS Metadata

The metadata is a map-tree of strings similar to this (YMMV):

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
