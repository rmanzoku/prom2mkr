package mpp2m

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

var ignoreNames = map[string]bool{
	"go_info": true,
}
var ignoreLabels = map[string]bool{
	"error": true,
}

// Prom2mkrPlugin mackerel plugin for Prometheus metrics
type Prom2mkrPlugin struct {
	Prefix       string
	URL          string
	Keys         []string
	GraphDefFile string
}

func (p Prom2mkrPlugin) traverseMap(families []*prom2json.Family) (map[string]float64, error) {
	stat := make(map[string]float64)
	var err error
	var name string

	for _, f := range families {

		_, ok := ignoreNames[f.Name]
		if ok {
			continue
		}

		name = strings.Replace(f.Name, "_", ".", -1)

		switch f.Type {
		case "COUNTER":
			for _, m := range f.Metrics {
				mm := m.(prom2json.Metric)
				n := name

				if len(mm.Labels) == 0 {
					stat[n], err = strconv.ParseFloat(mm.Value, 64)
					continue
				}

				for k, l := range mm.Labels {
					_, ok := ignoreLabels[k]
					if ok {
						continue
					}

					n = n + "." + k + "_" + l
				}

				stat[n], err = strconv.ParseFloat(mm.Value, 64)
				if err != nil {
					return nil, err
				}
			}

		case "GAUGE":
			for _, m := range f.Metrics {
				mm := m.(prom2json.Metric)
				n := name

				for k, l := range mm.Labels {
					_, ok := ignoreLabels[k]
					if ok {
						continue
					}

					n = n + "." + k + "_" + l
				}

				stat[n], err = strconv.ParseFloat(mm.Value, 64)
				if err != nil {
					return nil, err
				}
			}

		case "SUMMARY":
			for _, m := range f.Metrics {
				ss := m.(prom2json.Summary)
				n := name

				for k, l := range ss.Labels {
					_, ok := ignoreLabels[k]
					if ok {
						continue
					}

					n = n + "." + k + "_" + l
				}

				for k, q := range ss.Quantiles {
					quantile := strings.Replace(k, ".", "_", -1)
					stat[n+"."+quantile], err = strconv.ParseFloat(q, 64)
					if err != nil {
						return nil, err
					}
				}

				stat[n+".count"], err = strconv.ParseFloat(ss.Count, 64)
				if err != nil {
					return nil, err
				}

				stat[n+".sum"], err = strconv.ParseFloat(ss.Sum, 64)
				if err != nil {
					return nil, err
				}
			}

		default:
			fmt.Println(f.Type)
		}

	}

	return stat, nil
}

func (p Prom2mkrPlugin) fetchFamilies() ([]*prom2json.Family, error) {

	mfChan := make(chan *dto.MetricFamily, 1024)

	go func() {
		err := prom2json.FetchMetricFamilies(p.URL, mfChan, "", "", true)
		if err != nil {
			panic(err)
		}
	}()

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	return result, nil
}

// FetchMetrics interface for mackerelplugin
func (p Prom2mkrPlugin) FetchMetrics() (map[string]float64, error) {

	result, err := p.fetchFamilies()
	if err != nil {
		return nil, err
	}

	ret, err := p.traverseMap(result)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (p Prom2mkrPlugin) createGraphDef() mp.GraphDef {
	graphs := map[string]mp.Graphs{}
	families, err := p.fetchFamilies()
	if err != nil {
		panic(err)
	}

	for _, f := range families {
		name := strings.Replace(f.Name, "_", ".", -1)
		unit := "float"
		switch f.Type {
		case "COUNTER":
			unit = "integer"
		case "GAUGE":
			unit = "integer"
		case "HISTOGRAM":
			unit = "integer"
		case "SUMMERY":
			unit = "integer"
		}

		graphs[name] = mp.Graphs{
			Unit: unit,
		}
	}

	return mp.GraphDef{Graphs: graphs}
}

// GraphDefinition aa
func (p Prom2mkrPlugin) GraphDefinition() map[string]mp.Graphs {
	ret := mp.GraphDef{}

	_, err := os.Stat(p.GraphDefFile)
	if !os.IsNotExist(err) {
		file, err := ioutil.ReadFile(p.GraphDefFile)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(file, &ret)
		if err != nil {
			panic(err)
		}

		return ret.Graphs
	}

	ret = p.createGraphDef()
	file, err := os.Create(p.GraphDefFile)
	if err != nil {
		panic(err)
	}

	jsonByte, err := json.Marshal(ret)
	if err != nil {
		panic(err)
	}

	_, err = file.Write(jsonByte)
	if err != nil {
		panic(err)
	}

	return ret.Graphs
}

// Do the plugin
func Do() {
	var (
		optPrefix       = flag.String("metric-key-prefix", "", "Metric key prefix")
		optURL          = flag.String("url", "", "The bind url to use for the control server")
		optTempfile     = flag.String("tempfile", "", "Temp file name")
		optTempGraphDef = flag.String("tempGraphDef", "/tmp/prom2mkr.json", "Temp file for GraphDefinition")
	)
	flag.Parse()

	var p2m Prom2mkrPlugin
	p2m.Prefix = *optPrefix
	p2m.URL = *optURL
	p2m.GraphDefFile = *optTempGraphDef

	helper := mp.NewMackerelPlugin(p2m)
	helper.Tempfile = *optTempfile
	helper.Run()
}
