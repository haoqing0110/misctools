// Program stockopt optimizes a stock sale subject to limitations of capital
// gains.  The input to the program is an .xls spreadsheet as generated from
// the Gain/Loss view of the MSSB stock plan site.
//
// The output is a table listing how many of each lot of stock should be sold,
// the total sale price based on the estimated sale values from MSSB, and the
// total capital gain from the sale.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"bitbucket.org/creachadair/misctools/stockopt/currency"
	"bitbucket.org/creachadair/misctools/stockopt/solver"
	"bitbucket.org/creachadair/misctools/stockopt/statement"
)

var (
	inputPath    = flag.String("input", "", "Input file (.xls or .csv)")
	ageMonths    = flag.Int("age", 12, "Minimum age in months (12 months is the short-term cutoff)")
	planFilter   = flag.String("plan", "GSU Class C", "Consider only shares issued under this plan")
	capGainLimit = flag.String("gain", "$25000", "Capital gain limit in USD")
	printSummary = flag.Bool("summary", false, "Print summary of available shares")
	writeCSV     = flag.String("write", "", "Write input data as CSV to this file")
	allowLoss    = flag.Bool("loss", false, "Allow sale of capital losses")
)

func main() {
	flag.Parse()
	if *inputPath == "" {
		log.Fatal("You must provide an -input .xls path")
	}

	// Convert the capital gains cap into a currency value.
	maxGain, err := currency.ParseUSD(*capGainLimit)
	if err != nil {
		log.Fatalf("Invalid cap %q: %v", *capGainLimit, err)
	}

	// Read and parse the input spreadsheet, filtering out entries with 0
	// available shares, those issued more recently than the specified age, and
	// not matching the specified plan filter.
	data, err := ioutil.ReadFile(*inputPath)
	if err != nil {
		log.Fatalf("Reading statement: %v", err)
	}
	parse := statement.ParseXLS
	if filepath.Ext(*inputPath) == ".csv" {
		parse = statement.ParseCSV
	}

	then := time.Now().AddDate(0, -*ageMonths, 0)
	es, err := parse(data, func(e *statement.Entry) bool {
		return e.Available > 0 && e.Acquired.Before(then) &&
			(*planFilter == "" || e.Plan == *planFilter) &&
			(e.Gain >= 0 || *allowLoss)
	})
	if err != nil {
		log.Fatalf("Parsing statement: %v", err)
	}

	// Compute the total value of the portfolio, just for cosmetics.
	var totalValue, totalGain currency.Value
	var totalShares int
	for _, e := range es {
		totalShares += e.Available
		v := currency.Value(e.Available)
		totalValue += v * e.Price
		totalGain += v * e.Gain
	}

	fmt.Printf(`Input file:   %q
Minimum age:  %d months
Gains cap:    %s
Allow loss:   %v
Total shares: %d
Total value:  %s
Total gains:  %s

`, *inputPath, *ageMonths, maxGain.USD(), *allowLoss, totalShares, totalValue.USD(), totalGain.USD())

	// If requested, print a summary of available shares.
	if *printSummary {
		fmt.Println("Available shares:")
		for _, e := range es {
			fmt.Printf("%2d. %s\n", e.Index, e.Format(-1))
		}
		fmt.Println()
	}
	// If requested, write entries as CSV.
	if *writeCSV != "" {
		f, err := os.Create(*writeCSV)
		if err != nil {
			log.Fatalf("Creating CSV file: %v", err)
		}
		err = statement.WriteCSV(es, f)
		cerr := f.Close()
		if err != nil {
			log.Fatalf("Writing CSV: %v", err)
		} else if cerr != nil {
			log.Fatalf("Closing CSV file: %v", err)
		}
	}
	solve(es, maxGain)
}

func solve(es []*statement.Entry, maxGain currency.Value) {
	soln := solver.New(es2e(es)).Solve(maxGain)
	sort.Slice(soln, func(i, j int) bool {
		return statement.EntryLess(soln[i].ID.(*statement.Entry), soln[j].ID.(*statement.Entry))
	})

	var soldValue, soldGains currency.Value
	var soldShares int
	for _, elt := range soln {
		e := elt.ID.(*statement.Entry)
		soldShares += elt.N
		soldValue += currency.Value(elt.N) * elt.Value
		soldGains += currency.Value(elt.N) * elt.Gain
		fmt.Printf("Sell [lot %2d]: %s\n", e.Index, e.Format(elt.N))
	}
	fmt.Printf("\nSold shares:  %d\nSold value:   %s\nSold gains:   %s\n",
		soldShares, soldValue.USD(), soldGains.USD())
}

// es2e converts statement entries to solver entries.
func es2e(es []*statement.Entry) []solver.Entry {
	out := make([]solver.Entry, len(es))
	for i, e := range es {
		out[i] = solver.Entry{
			ID:    e,
			N:     e.Available,
			Value: e.Price,
			Gain:  e.Gain,
		}
	}
	return out
}
