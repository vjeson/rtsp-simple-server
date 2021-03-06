package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	metricsAddress = ":9998"
)

type metricsData struct {
	countClient    int64
	countPublisher int64
	countReader    int64
}

type metricsGatherReq struct {
	res chan *metricsData
}

type metrics struct {
	p        *program
	listener net.Listener
	mux      *http.ServeMux
	server   *http.Server
}

func newMetrics(p *program) (*metrics, error) {
	listener, err := net.Listen("tcp", metricsAddress)
	if err != nil {
		return nil, err
	}

	m := &metrics{
		p:        p,
		listener: listener,
	}

	m.mux = http.NewServeMux()
	m.mux.HandleFunc("/metrics", m.onMetrics)

	m.server = &http.Server{
		Handler: m.mux,
	}

	m.p.log("[metrics] opened on " + metricsAddress)
	return m, nil
}

func (m *metrics) run() {
	err := m.server.Serve(m.listener)
	if err != http.ErrServerClosed {
		panic(err)
	}
}

func (m *metrics) close() {
	m.server.Shutdown(context.Background())
}

func (m *metrics) onMetrics(w http.ResponseWriter, req *http.Request) {
	res := make(chan *metricsData)
	m.p.metricsGather <- metricsGatherReq{res}
	data := <-res

	if data == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	out := ""
	now := time.Now().UnixNano() / 1000000

	out += fmt.Sprintf("clients %d %v\n", data.countClient, now)
	out += fmt.Sprintf("publishers %d %v\n", data.countPublisher, now)
	out += fmt.Sprintf("readers %d %v\n", data.countReader, now)

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, out)
}
