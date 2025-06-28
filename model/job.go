package model

import (
	"encoding/json"
	"fmt"
)

type MailJob struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}


func UnmarshalMailJob(data []byte) (*MailJob, error) {
	var job MailJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mail job: %w", err)
	}
	return &job, nil
}

func (j *MailJob) Marshal() ([]byte, error) {
	data, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mail job: %w", err)
	}
	return data, nil
}
