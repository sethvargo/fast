package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

var userAgent = fmt.Sprintf("Fast/%s (+%s; %s)",
	projectVersion, projectURL, runtime.Version())

// Downloader is the collection of pages to download, measure, and aggregate.
type Downloader struct {
	urls      []string
	metricsCh chan *DownloadMetric
}

type NewDownloaderInput struct {
	URLs []string
}

type DownloadMetric struct {
	duration time.Duration
	bits     int
}

func (m *DownloadMetric) Append(other *DownloadMetric) {
	m.duration += other.duration
	m.bits += other.bits
}

// Rate returns the total bits per second.
func (m *DownloadMetric) Rate() float64 {
	return float64(m.bits) / float64(m.duration.Seconds())
}

func (m *DownloadMetric) String() string {
	rate := m.Rate()

	var unit string
	switch {
	case rate > EB:
		rate = rate / EB
		unit = "EBps"
	case rate > PB:
		rate = rate / PB
		unit = "Pbps"
	case rate > TB:
		rate = rate / TB
		unit = "Tbps"
	case rate > GB:
		rate = rate / GB
		unit = "Gbps"
	case rate > MB:
		rate = rate / MB
		unit = "Mbps"
	case rate > KB:
		rate = rate / KB
		unit = "Kbps"
	default:
		unit = "Bps"
	}

	return fmt.Sprintf("%.2f %s", rate, unit)
}

func NewDownloader(i *NewDownloaderInput) (*Downloader, error) {
	var d Downloader
	d.urls = i.URLs
	d.metricsCh = make(chan *DownloadMetric, 500)

	return &d, nil
}

func (d *Downloader) DownloadAll() (*DownloadMetric, error) {
	defer close(d.metricsCh)

	finalMetric := &DownloadMetric{}

	for _, u := range d.urls {
		metric, err := d.Download(u)
		if err != nil {
			return nil, err
		}
		finalMetric.Append(metric)
	}

	return finalMetric, nil
}

// Download downloads a single url
func (d *Downloader) Download(u string) (*DownloadMetric, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request failed")
	}

	// Set headers so we look like a browser
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://fast.com/")
	req.Header.Set("Origin", "https://fast.com")

	// Start the clock
	t1 := time.Now()

	resp, err := d.client().Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "starting request failed")
	}
	defer resp.Body.Close()

	// A buffer to hold our data
	var b bytes.Buffer

	// Create our doneCh - this will be used to stop the routine
	doneCh := make(chan struct{}, 1)

	// Periodically update the parent about the current download stats
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-time.After(250 * time.Millisecond):
				metric := &DownloadMetric{
					duration: time.Since(t1),
					bits:     b.Len() * 8,
				}

				select {
				case <-doneCh:
					return
				case d.metricsCh <- metric:
				default:
				}
			}
		}
	}()

	// Stop the update routine when we are done
	defer close(doneCh)

	// Copy into the buffer - this will block while "downloading"
	if _, err := io.Copy(&b, resp.Body); err != nil {
		return nil, errors.Wrap(err, "download failed")
	}

	// Create our final metric
	metric := &DownloadMetric{
		duration: time.Since(t1),
		bits:     b.Len() * 8,
	}

	return metric, nil
}

func (d *Downloader) MetricsCh() <-chan *DownloadMetric {
	return d.metricsCh
}

func (d *Downloader) client() *http.Client {
	var h http.Client
	return &h
}

// mkfifo /tmp/fast.com.test.fifo ;token=$(curl -s https://fast.com/app-ed402d.js|egrep -om1 'token:"[^"]+'|cut -f2 -d'"'); curl -s "https://api.fast.com/netflix/speedtest?https=true&token=$token" |egrep -o 'https[^"]+'|while read url; do first=${url/speedtest/speedtest\/range\/0-2048}; next=${url/speedtest/speedtest\/range\/0-26214400};(curl -s -H 'Referer: https://fast.com/' -H 'Origin: https://fast.com' "$first" > /tmp/fast.com.test.fifo; for i in {1..10}; do curl -s -H 'Referer: https://fast.com/' -H 'Origin: https://fast.com'  "$next">>/tmp/fast.com.test.fifo; done)& done & pv /tmp/fast.com.test.fifo > /dev/null; rm /tmp/fast.com.test.fifo
