// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code is a prototype and not engineered for production use.
// Error handling is incomplete or inappropriate for usage beyond
// a development sample.

package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	var (
		outputType          string
		txtDomain           string
		variable            string
		namePrefix          string
		description         string
		tags                string
		priority            string
		nextHopType         string
		nextHopValue        string
		nextHopInstanceZone string
	)

	app := cli.NewApp()
	app.Name = "txt2route"
	app.Usage = "download DNS TXT entries into terraform routes"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "output, o",
			Value:       "routes",
			Usage:       "output type: tfvars|variables|routes",
			Destination: &outputType,
		},
		cli.StringFlag{
			Name:        "domain",
			Value:       "_spf.google.com",
			Usage:       "TXT domain to use for lookup",
			Destination: &txtDomain,
		},
		cli.StringFlag{
			Name:        "name",
			Value:       "google_netblock_cidrs",
			Usage:       "name to use for variable in output",
			Destination: &variable,
		},
		cli.StringFlag{
			Name:        "route-prefix",
			Value:       "google-route",
			Usage:       "[route only] prefix for route name",
			Destination: &namePrefix,
		},
		cli.StringFlag{
			Name:        "route-description",
			Value:       "google private access netblock from _spf.google.com",
			Usage:       "[route only] route description",
			Destination: &description,
		},
		cli.StringFlag{
			Name:        "route-tags",
			Value:       "",
			Usage:       "[route only] tags (i.e. [ \"foo\", \"bar\" ]), \"\" for no tags",
			Destination: &tags,
		},
		cli.StringFlag{
			Name:        "route-priority",
			Value:       "50",
			Usage:       "[route only] route priority",
			Destination: &priority,
		},
		cli.StringFlag{
			Name:        "route-hop-type",
			Value:       "next_hop_internet",
			Usage:       "[route only] type of next hop",
			Destination: &nextHopType,
		},
		cli.StringFlag{
			Name:        "route-hop-value",
			Value:       "true",
			Usage:       "[route only] value for next hop",
			Destination: &nextHopValue,
		},
		cli.StringFlag{
			Name:        "route-instance-zone",
			Value:       "",
			Usage:       "[route only] instance zone (if applicable)",
			Destination: &nextHopInstanceZone,
		},
	}

	app.Action = func(c *cli.Context) error {
		cidrs := lookup(txtDomain)
		var result string
		switch strings.ToLower(outputType) {
		case "tfvars":
			result = tfvars(variable, cidrs)
		case "variables":
			result = variables(variable, cidrs)
		case "routes":
			result = routes(variable, cidrs, namePrefix, description,
				tags, priority, nextHopType, nextHopValue, nextHopInstanceZone)
		default:
			return errors.New(fmt.Sprintf("Unknown output type: %s", outputType))
		}
		fmt.Println(result)
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func lookup(startDomain string) []string {
	var items sync.WaitGroup
	// TODO: deadlocks if buffer isn't high enough, shouldn't just use high number of 20 here
	lookups := make(chan string, 20)
	entries := make(chan string, 20)
	ipv4 := make([]string, 0)
	// TODO: add ipv6 (when needed?)
	// ipv6 := make([]string, 0)

	go func() {
		for domain := range lookups {
			txtrecords, err := net.LookupTXT(domain)
			if err != nil {
				log.Fatalf("error looking up TXT %s: %v\n", domain, err)
			}
			for _, record := range txtrecords {
				for _, entry := range strings.Split(record, " ") {
					entries <- entry
					items.Add(1)
				}
			}
			items.Done()
		}
	}()

	go func() {
		for entry := range entries {
			parts := strings.Split(entry, ":")
			switch parts[0] {
			case "ip4":
				ipv4 = append(ipv4, parts[1])
			case "ip6":
				// ipv6 = append(ipv6, strings.Join(parts[1:], ":"))
			case "include":
				lookups <- parts[1]
				items.Add(1)
			}
			items.Done()
		}
	}()

	lookups <- startDomain
	items.Add(1)
	items.Wait()
	return ipv4
}

func tfvars(variable string, cidrs []string) string {
	// Print cidrs as variable list
	result := fmt.Sprintf("%s = [\n", variable)
	for _, cidr := range cidrs {
		result += fmt.Sprintf("\t\t\"%s\",\n", cidr)
	}
	result += fmt.Sprintf("\t]\n")
	return result
}

func variables(variable string, cidrs []string) string {
	// Print variable with routes list as default
	result := fmt.Sprintf("variable \"%s\" { default = [\n", variable)
	for _, cidr := range cidrs {
		result += fmt.Sprintf("\t\t\"%s\",\n", cidr)
	}
	result += fmt.Sprintf("\t] }\n")
	return result
}

func routes(
	variable string,
	cidrs []string,
	namePrefix string,
	description string,
	tags string,
	priority string,
	nextHopType string,
	nextHopValue string,
	nextHopInstanceZone string) string {

	nextHopInstanceZoneAddendum := ""
	if nextHopType == "next_hop_instance" && nextHopInstanceZone != "" {
		nextHopInstanceZoneAddendum = fmt.Sprintf("\n\tnext_hop_instance_zone\t\t\t = \"%s\"\n", nextHopInstanceZone)
	}
	tagsAddendum := ""
	if tags != "" {
		tagsAddendum = fmt.Sprintf("\n\ttags\t\t  = %s\n", tags)
	}

	result := fmt.Sprintf("variable \"%s\" { default = [\n", variable)
	for i, cidr := range cidrs {
		result += fmt.Sprintf(`
{
	name                   = "%s-%d"
	description            = "%s"
	destination_range      = "%s"%s
	priority               = "%s"
	%s      = "%s"%s
},`, namePrefix, i, description, cidr, tagsAddendum, priority, nextHopType, nextHopValue, nextHopInstanceZoneAddendum)
	}
	result += fmt.Sprintf("\t ] }\n")
	return result
}
