# txt2route
dns txt spf records to terraform list/routes

# usage

      NAME:
         txt2route - download DNS TXT entries into terraform routes

      USAGE:
         txt2route [global options] command [command options] [arguments...]

      VERSION:
         0.0.0

      COMMANDS:
         help, h  Shows a list of commands or help for one command

      GLOBAL OPTIONS:
         --output value, -o value     output type: tfvars|variables|routes (default: "routes")
         --domain value               TXT domain to use for lookup (default: "_spf.google.com")
         --name value                 name to use for variable in output (default: "google_netblock_cidrs")
         --route-prefix value         [route only] prefix for route name (default: "google-route")
         --route-description value    [route only] route description (default: "google private access netblock from _spf.google.com")
         --route-tags value           [route only] tags (i.e. [ "foo", "bar" ]), "" for no tags
         --route-priority value       [route only] route priority (default: "50")
         --route-hop-type value       [route only] type of next hop (default: "next_hop_internet")
         --route-hop-value value      [route only] value for next hop (default: "true")
         --route-instance-zone value  [route only] instance zone (if applicable)
         --help, -h                   show help
         --version, -v                print the version

