package metronome

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dcos/dcos-cli/pkg/dcos"
	"github.com/dcos/dcos-cli/pkg/httpclient"
)

// Client is a client for Cosmos.
type Client struct {
	http *httpclient.Client
}

// JobsOption is a functional Option to set the `embed` query parameters
type JobsOption func(query url.Values)

// EmbedActiveRun sets the `embed`option to activeRuns
func EmbedActiveRun() JobsOption {
	return func(query url.Values) {
		query.Add("embed", "activeRuns")
	}
}

// EmbedSchedule sets the `embed`option to schedules
func EmbedSchedule() JobsOption {
	return func(query url.Values) {
		query.Add("embed", "schedules")
	}
}

// EmbedHistory sets the `embed`option to history
func EmbedHistory() JobsOption {
	return func(query url.Values) {
		query.Add("embed", "history")
	}
}

// EmbedHistorySummary sets the `embed`option to historySummary
func EmbedHistorySummary() JobsOption {
	return func(query url.Values) {
		query.Add("embed", "historySummary")
	}
}

// NewClient creates a new Metronome client.
func NewClient(baseClient *httpclient.Client) *Client {
	return &Client{
		http: baseClient,
	}
}

// Jobs returns a list of all job definitions.
func (c *Client) Jobs(opts ...JobsOption) ([]Job, error) {

	req, err := c.http.NewRequest("GET", "/v1/jobs", nil, httpclient.FailOnErrStatus(true))
	if err != nil {
		return nil, err
	}

	// Add embed query parameters to the request URL
	q := req.URL.Query()
	for _, opt := range opts {
		opt(q)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jobs []Job
	err = json.NewDecoder(resp.Body).Decode(&jobs)

	return jobs, err
}

func (c *Client) addOrUpdateJob(job *Job, add bool) (*Job, error) {
	jsonBytes, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	buf := bytes.NewBuffer(jsonBytes)
	if add {
		req, err = c.http.NewRequest("POST", "/v1/jobs", buf)
	} else {
		req, err = c.http.NewRequest("PUT", "/v1/jobs/"+job.ID, buf)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 201:
		var j Job
		if err = json.NewDecoder(resp.Body).Decode(&j); err != nil {
			return nil, err
		}
		return &j, nil
	default:
		var apiError *Error
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			return nil, err
		}
		apiError.Code = resp.StatusCode
		return nil, apiError
	}
}

// AddJob creates a job.
func (c *Client) AddJob(job *Job) (*Job, error) {
	return c.addOrUpdateJob(job, true)
}

// UpdateJob updates an existing job.
func (c *Client) UpdateJob(job *Job) (*Job, error) {
	return c.addOrUpdateJob(job, false)
}

// RunJob triggers a run of the job with a given runID right now.
func (c *Client) RunJob(runID string) (*Run, error) {
	resp, err := c.http.Post("/v1/jobs/"+runID+"/runs", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 201:
		var run Run
		if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
			return nil, err
		}
		return &run, nil
	case 404:
		return nil, fmt.Errorf("job %s does not exist", runID)
	default:
		var apiError *dcos.Error
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			return nil, err
		}
		return nil, apiError
	}
}

// RemoveJob removes a job.
func (c *Client) RemoveJob(jobID string) error {
	resp, err := c.http.Delete("/v1/jobs/" + jobID)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return nil
	case 409:
		return fmt.Errorf("job %s is running", jobID)
	default:
		var apiError *dcos.Error
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			return err
		}
		return apiError
	}
}

// AddSchedule adds a schedule to the job with the given jobID.
func (c *Client) AddSchedule(jobID string, schedule Schedule) (*Schedule, error) {
	jsonBytes, err := json.Marshal(schedule)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	buf := bytes.NewBuffer(jsonBytes)

	req, err = c.http.NewRequest("POST", "/v1/jobs/"+jobID+"/schedules", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 201:
		var s Schedule
		if err = json.NewDecoder(resp.Body).Decode(&s); err != nil {
			return nil, err
		}
		return &s, nil
	default:
		var apiError *Error
		if err := json.NewDecoder(resp.Body).Decode(&apiError); err != nil {
			return nil, err
		}
		apiError.Code = resp.StatusCode
		return nil, apiError
	}
}
