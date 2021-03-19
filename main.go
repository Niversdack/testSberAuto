package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	openParenthesis  = "("
	closeParenthesis = ")"
	openCurly        = "{"
	closeCurly       = "}"
	openSquare       = "["
	closeSquare      = "]"
)

// StringService provides operations on strings.
type StringService interface {
	Validate(string) (string, error)
	Fix(string) (string, error)
}

// stringService is a concrete implementation of StringService
type stringService struct{}

func (stringService) Validate(s string) (string, error) {
	if s == "" {
		return "", ErrEmpty
	}
	list := make([]string, 0)
	var err error
	a := strings.Split(s, "")
	for _, r := range a {
		switch r {
		case openParenthesis, openCurly, openSquare:
			list = append(list, r)
		case closeParenthesis:
			err = ValidateBracked(&list, openParenthesis)
			if err != nil {
				return err.Error(), nil
			}
		case closeCurly:
			err = ValidateBracked(&list, openCurly)
			if err != nil {
				return err.Error(), nil
			}
		case closeSquare:
			err = ValidateBracked(&list, openSquare)
			if err != nil {
				return err.Error(), nil
			}
		default:
			return "Not Balanced", ErrBadString
		}
	}
	result := len(list)
	if result == 0 {
		return "Balanced", nil
	}
	return "Not Balanced", nil
}
func ValidateBracked(list *[]string, bracketOpen string) error {
	stack := *list
	if stack[len(stack)-1] == bracketOpen {
		*list = stack[:len(stack)-1]
		return nil
	}

	return errors.New("Not Balanced")
}

func (stringService) Fix(s string) (string, error) {
	list := make([]string, 0)
	a := strings.Split(s, "")
	result := ""

	for _, r := range a {

		switch r {
		case openParenthesis, openCurly, openSquare:
			list = append(list, r)
			result += r
		case closeParenthesis:
			result += FixBracked(&list, openParenthesis, closeParenthesis)
		case closeCurly:
			result += FixBracked(&list, openCurly, closeCurly)
		case closeSquare:
			result += FixBracked(&list, openSquare, closeSquare)
		default:
			return "", ErrBadString
		}
	}
	if len(list) > 0 {
		for _, s2 := range list {
			switch s2 {
			case openParenthesis:
				result += closeParenthesis
			case openCurly:
				result += closeCurly
			case openSquare:
				result += closeSquare
			}
		}
	}
	return result, nil
}
func FixBracked(list *[]string, bracketOpen string, bracketClose string) string {
	result := ""
	stack := *list
	if len(stack) > 0 {
		if stack[len(stack)-1] != bracketOpen {
			*list = append(stack, bracketOpen)
			result += bracketOpen
		}
		*list = stack[:len(stack)-1]
		result += bracketClose
	} else {
		*list = append(stack, bracketOpen)
		result += bracketOpen
		result += bracketClose
	}
	return result
}

// ErrEmpty is returned when an input string is empty.
var ErrEmpty = errors.New("empty string")
var ErrBadString = errors.New("bad string")

// For each method, we define request and response structs
type validateRequest struct {
	S string `json:"s"`
}

type validateResponse struct {
	V   string `json:"v"`
	Err string `json:"err,omitempty"` // errors don't define JSON marshaling
}

type fixRequest struct {
	S string `json:"s"`
}

type fixResponse struct {
	S   string `json:"v"`
	Err string `json:"err,omitempty"` // errors don't define JSON marshaling
}

// Endpoints are a primary abstraction in go-kit. An endpoint represents a single RPC (method in our service interface)
func makeValidateEndpoint(svc StringService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		opsRequest.Inc()
		req := request.(validateRequest)
		v, err := svc.Validate(req.S)
		if err != nil {
			if err == ErrBadString {
				return validateResponse{req.S, err.Error()}, nil
			}
			return validateResponse{v, err.Error()}, nil
		}
		return validateResponse{v, ""}, nil
	}
}

func makeFixEndpoint(svc StringService) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		opsRequest.Inc()
		req := request.(fixRequest)
		v, err := svc.Fix(req.S)
		if err != nil {
			return fixResponse{req.S, err.Error()}, nil
		}

		return fixResponse{v, ""}, nil

	}
}
func main() {
	svc := stringService{}

	validateHandler := httptransport.NewServer(
		makeValidateEndpoint(svc),
		decodeUppercaseRequest,
		encodeResponse,
	)

	countHandler := httptransport.NewServer(
		makeFixEndpoint(svc),
		decodeCountRequest,
		encodeResponse,
	)
	recordMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/validate", validateHandler)
	http.Handle("/fix", countHandler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
	opsRequest = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_total_requests",
		Help: "The total number requests",
	})
)

func decodeUppercaseRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request validateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func decodeCountRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request fixRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil

}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
