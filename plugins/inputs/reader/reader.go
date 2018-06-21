package reader

import (
	"io/ioutil"
	"log"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal/globpath"
	"github.com/influxdata/telegraf/plugins/parsers"
)

type Reader struct {
	Filepaths     []string `toml:"files"`
	FromBeginning bool
	DataFormat    string `toml:"data_format"`
	ParserConfig  parsers.Config
	Parser        parsers.Parser
	Tags          []string

	Filenames []string
}

const sampleConfig = `## Log files to parse.
## These accept standard unix glob matching rules, but with the addition of
## ** as a "super asterisk". ie:
##   /var/log/**.log     -> recursively find all .log files in /var/log
##   /var/log/*/*.log    -> find all .log files with a parent dir in /var/log
##   /var/log/apache.log -> only tail the apache log file
files = ["/var/log/apache/access.log"]

## The dataformat to be read from files
## Each data format has its own unique set of configuration options, read
## more about them here:
## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
data_format = ""
'''`

// SampleConfig returns the default configuration of the Input
func (r *Reader) SampleConfig() string {
	return sampleConfig
}

func (r *Reader) Description() string {
	return "reload and gather from file[s] on telegraf's interval"
}

func (r *Reader) Gather(acc telegraf.Accumulator) error {
	r.refreshFilePaths()
	for _, k := range r.Filenames {
		metrics, err := r.readMetric(k)
		if err != nil {
			return err
		}

		for _, m := range metrics {
			acc.AddFields(m.Name(), m.Fields(), m.Tags())
		}
	}
	return nil
}

func (r *Reader) compileParser() {
	if r.DataFormat == "grok" {
		log.Printf("Grok isn't supported yet")
		return
	}
	r.ParserConfig = parsers.Config{
		DataFormat: r.DataFormat,
		TagKeys:    r.Tags,
	}
	nParser, err := parsers.NewParser(&r.ParserConfig)
	if err != nil {
		log.Printf("E! Error building parser: %v", err)
	}

	r.Parser = nParser
}

func (r *Reader) refreshFilePaths() {
	var allFiles []string
	for _, filepath := range r.Filepaths {
		g, err := globpath.Compile(filepath)
		if err != nil {
			log.Printf("E! Error Glob %s failed to compile, %s", filepath, err)
			continue
		}
		files := g.Match()

		for k := range files {
			allFiles = append(allFiles, k)
		}
	}

	r.Filenames = allFiles
}

//requires that Parser has been compiled
func (r *Reader) readMetric(filename string) ([]telegraf.Metric, error) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("E! File could not be opened: %v", filename)
	}

	return r.Parser.Parse(fileContents)

}