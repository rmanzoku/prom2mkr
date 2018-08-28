package mpp2m

import (
	"flag"
	"fmt"
	"log"
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

func (p Prom2mkrPlugin) traverseMap(families []*prom2json.Family, prefix string) (map[string]float64, error) {
	stat := make(map[string]float64)
	var err error
	var name string

	for _, f := range families {
		if prefix != "" {
			name = prefix + "." + strings.Replace(f.Name, "_", ".", -1)
		} else {
			name = strings.Replace(f.Name, "_", ".", -1)
		}

		switch f.Type {
		case "COUNTER":
			for _, m := range f.Metrics {
				mm := m.(prom2json.Metric)

				if len(mm.Labels) == 0 {
					stat[name], err = strconv.ParseFloat(mm.Value, 64)
					continue
				}

				for k, l := range mm.Labels {
					n := name + "." + k + "_" + l
					stat[n], err = strconv.ParseFloat(mm.Value, 64)

					if err != nil {
						return nil, err
					}
				}
			}

		case "GAUGE":
			for _, m := range f.Metrics {
				mm := m.(prom2json.Metric)
				if len(mm.Labels) == 0 {
					stat[name], err = strconv.ParseFloat(mm.Value, 64)
					continue
				}

				for k, l := range mm.Labels {
					n := name + "." + k + "_" + l
					stat[n], err = strconv.ParseFloat(mm.Value, 64)

					if err != nil {
						return nil, err
					}
				}
			}

		// case "SUMMERY":
		// 	f.Metrics[0].(prom2json.Summary)
		// 	if err != nil {
		// 		return nil, err
		// 	}

		default:
			fmt.Println(f.Type)
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

	ret, err := p.traverseMap(result, p.Prefix)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p Prom2mkrPlugin) MetricKeyPrefix() string {
	return p.Prefix
}

// Do the plugin
func Do() {
	var (
		optPrefix = flag.String("metric-key-prefix", "", "Metric key prefix")
		optURL    = flag.String("url", "", "The bind url to use for the control server")
		// optTempfile = flag.String("tempfile", "", "Temp file name")
	)
	flag.Parse()

	var p2m Prom2mkrPlugin
	p2m.Prefix = *optPrefix
	p2m.URL = *optURL

	metrics, err := p2m.FetchMetrics()
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now().Unix()
	for k, v := range metrics {
		fmt.Printf("%s\t%f\t%d\n", k, v, now)
	}

	// helper := mp.NewMackerelPlugin(p2m)
	// helper.Tempfile = *optTempfile
	// helper.Run()
}
