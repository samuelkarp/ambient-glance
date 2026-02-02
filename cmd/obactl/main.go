/*
   Copyright 2025 Google LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	oba "github.com/OneBusAway/go-sdk"
	"github.com/OneBusAway/go-sdk/option"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "obactl",
		Short: "obactl is a tool for querying OneBusAway",
	}
	rootCmd.AddCommand(stopsCommand())
	rootCmd.AddCommand(arrivalCommand())
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func stopsCommand() *cobra.Command {
	stops := &cobra.Command{
		Use:   "stops",
		Short: "Finds stops for a given location",
	}
	var (
		lat float64
		lon float64
	)
	flags := stops.Flags()
	flags.Float64Var(&lat, "lat", 0, "Latitude to search for stops for")
	flags.Float64Var(&lon, "lon", 0, "Longitude to search for stops for")
	stops.MarkFlagRequired("lat")
	stops.MarkFlagRequired("lon")
	stops.RunE = func(cmd *cobra.Command, args []string) error {
		client := oba.NewClient(
			option.WithAPIKey("TEST"),
		)

		res, err := client.StopsForLocation.List(stops.Context(), oba.StopsForLocationListParams{
			Lat: oba.F(lat),
			Lon: oba.F(lon),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		if res == nil {
			return errors.New("nil response")
		}
		fmt.Println("AGENCIES")
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
		fmt.Fprintln(tw, "ID\tNAME")
		for _, a := range res.Data.References.Agencies {
			fmt.Fprintf(tw, "%s\t%s\n", a.ID, a.Name)
		}
		tw.Flush()
		fmt.Println()
		fmt.Println("ROUTES")
		tw = tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
		fmt.Fprintln(tw, "ID\tSHORT\tNAME")
		shorts := make(map[string]string)
		for _, r := range res.Data.References.Routes {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", r.ID, r.ShortName, r.LongName)
			shorts[r.ID] = r.ShortName
		}
		tw.Flush()

		fmt.Println()
		fmt.Println("STOPS")
		tw = tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
		fmt.Fprintln(tw, "ID\tDIR\tNAME\tROUTES")
		for _, s := range res.Data.List {
			routes := make([]string, 0)
			for _, r := range s.RouteIDs {
				routes = append(routes, shorts[r])
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.ID, s.Direction, s.Name, strings.Join(routes, ","))
		}
		tw.Flush()

		return nil
	}
	return stops
}

func arrivalCommand() *cobra.Command {
	arrival := &cobra.Command{
		Use:   "arrival",
		Short: "Finds arrival for a given location",
	}
	var (
		stopID string
	)
	flags := arrival.Flags()
	flags.StringVarP(&stopID, "id", "i", "", "stop ID")
	arrival.MarkFlagRequired("id")
	arrival.RunE = func(cmd *cobra.Command, args []string) error {
		client := oba.NewClient(
			option.WithAPIKey("TEST"),
		)

		res, err := client.ArrivalAndDeparture.List(arrival.Context(), stopID, oba.ArrivalAndDepartureListParams{})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		if res == nil {
			return errors.New("nil response")
		}
		loc, err := time.LoadLocation("America/Los_Angeles")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		agencies := make(map[string]string)
		for _, a := range res.Data.References.Agencies {
			agencies[a.ID] = a.Name
		}
		routes := make(map[string]string)
		for _, r := range res.Data.References.Routes {
			routes[r.ID] = r.AgencyID
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
		fmt.Fprintln(tw, "ID\tSHORT\tPREDICTED ARRIVAL\tDEPARTURE\tHEADSIGN\tAGENCY")
		for _, a := range res.Data.Entry.ArrivalsAndDepartures {
			arr := time.UnixMilli(a.PredictedArrivalTime).In(loc)
			dep := time.UnixMilli(a.PredictedDepartureTime).In(loc)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", a.RouteID, a.RouteShortName, arr.Format(time.Kitchen), dep.Format(time.Kitchen), a.TripHeadsign, agencies[routes[a.RouteID]])
		}
		tw.Flush()
		return nil
	}
	return arrival
}
