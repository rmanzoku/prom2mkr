package mpp2m

import (
	"flag"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	// mp "github.com/mackerelio/go-mackerel-plugin"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/prom2json"
)

// Prom2mkrPlugin mackerel plugin for Prometheus metrics
type Prom2mkrPlugin struct {
	Prefix string
	URL    string
}

func (p Prom2mkrPlugin) traverseMap(content []*prom2json.Family, path []string) (map[string]float64, error) {
	stat := make(map[string]float64)
	var err error
	var name string

	for _, c := range content {
		name = strings.Replace(c.Name, "_", ".", -1)

		for _, m := range c.Metrics {

			switch reflect.TypeOf(m) {
			case reflect.TypeOf(prom2json.Metric{}):

				metric := m.(prom2json.Metric)
				for _, l := range metric.Labels {
					name = name + "." + l
				}

				stat[name], err = strconv.ParseFloat(metric.Value, 64)
				if err != nil {
					fmt.Println("err")
				}

			default:
				continue
			}

		}

	}

	return stat, err
}

// FetchMetrics interface for mackerelplugin
func (p Prom2mkrPlugin) FetchMetrics() (map[string]float64, error) {
	ret := make(map[string]float64)

	mfChan := make(chan *dto.MetricFamily, 1024)

	go func() {
		err := prom2json.FetchMetricFamilies(p.URL, mfChan, "", "", true)
		if err != nil {
			log.Fatal(err)
		}
	}()

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	ret, err := p.traverseMap(result, []string{p.Prefix})
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p Prom2mkrPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "p2m"
	}
	return p.Prefix
}

// Do the plugin
func Do() {
	var (
		optPrefix = flag.String("metric-key-prefix", "p2m", "Metric key prefix")
		optURL    = flag.String("url", "", "The bind url to use for the control server")
		// optTempfile = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var p2m Prom2mkrPlugin
	p2m.Prefix = *optPrefix
	p2m.URL = *optURL

	metrics, _ := p2m.FetchMetrics()
	now := time.Now().Unix()
	for k, v := range metrics {
		fmt.Printf("%s\t%f\t%d\n", k, v, now)
	}

	// helper := mp.NewMackerelPlugin(p2m)
	// helper.Tempfile = *optTempfile
	// helper.Run()
}
