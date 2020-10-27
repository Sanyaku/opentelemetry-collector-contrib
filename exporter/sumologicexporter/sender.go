package sumologicexporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type sender struct {
	cfg    *Config
	cl     *http.Client
	buffer []pdata.LogRecord
}

type filterator interface {
	FilterOut(attributes pdata.AttributeMap) map[string]string
}

func newSender(cl *http.Client, cfg *Config, f filterator, bufferSize int) *sender {
	return &sender{
		cfg:    cfg,
		cl:     cl,
		buffer: make([]pdata.LogRecord, 0, bufferSize),
	}
}

// Send sends logs using the provided fields and clears the buffer after each invocation.
// It returns number of dropped frames in case of an error and the error itself.
func (s *sender) Send(fields string) (int, error) {
	err := s.sendLogs(s.cfg.LogFormat, fields)
	if err != nil {
		return len(s.buffer), err
	}
	s.buffer = s.buffer[:0]
	return 0, nil
}

// AppendToBuffer appends a log record to the buffer.
func (s *sender) AppendToBuffer(lr pdata.LogRecord) {
	s.buffer = append(s.buffer, lr)
}

func (s *sender) sendLogs(format LogFormat, fields string) error {
	switch format {
	case TextFormat:
		return s.sendLogsTextFormat(fields)

	case JSONFormat:
		return s.sendLogsJSONFormat(fields)
		// TODO
		return nil

	default:
		return errors.New("Unexpected log format")
	}
}

func (s *sender) sendLogsTextFormat(fields string) error {
	var (
		body strings.Builder
		errs []error
	)

	for j := 0; j < len(buffer); j++ {
		err := s.appendAndSend(buffer[j].Body().StringVal(), &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := s.send(body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

func (s *sender) send(pipeline string, body io.Reader, fields string) error {
	req, err := http.NewRequest(http.MethodPost, s.cfg.URL, body)
	if err != nil {
		return err
	}

	req.Header.Add("X-Sumo-Client", s.cfg.Client)

	if len(s.cfg.SourceHost) > 0 {
		req.Header.Add("X-Sumo-Host", s.cfg.SourceHost)
	}

	if len(s.cfg.SourceName) > 0 {
		req.Header.Add("X-Sumo-Name", s.cfg.SourceName)
	}

	if len(s.cfg.SourceCategory) > 0 {
		req.Header.Add("X-Sumo-Category", s.cfg.SourceCategory)
	}

	if pipeline == LogsPipeline {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Sumo-Fields", fields)
	} else if pipeline == MetricsPipeline {
		// ToDo: Implement metrics pipeline
	} else {
		return errors.New("Unexpected pipeline")
	}

	_, err = s.cl.Do(req)
	// ToDo: Add retries mechanism
	if err != nil {
		return err
	}
	return nil
}

func (s *sender) sendLogsJSONFormat(fields string) error {
	body := strings.Builder{}
	var errs []error

	for j := 0; j < len(s.buffer); j++ {
		data := s.FilterOut(s.buffer[j].Attributes(), true)
		data[logKey] = s.buffer[j].Body().StringVal()
		nextLine, err := json.Marshal(data)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = s.appendAndSend(bytes.NewBuffer(nextLine).String(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := s.send(LogsPipeline, body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

func (s *sender) appendAndSend(line string, body *strings.Builder, fields string) error {
	var err error
	if body.Len() > 0 && body.Len()+len(line) > s.config.MaxRequestBodySize {
		err = s.send(body.String(), fields)
		body.Reset()
	}

	if body.Len() > 0 {
		// Do not add newline if the body is empty
		body.WriteString("\n")
	}

	body.WriteString(line)
	return err
}
