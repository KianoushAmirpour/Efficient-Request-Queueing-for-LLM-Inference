package domain

type Status string

const (
	StatusCreated Status = "created"

	StatusCompleted Status = "completed"

	StatusFailed Status = "failed"
)

func (j *Job) Complete() error {
	if j.Status != StatusCreated {
		return ErrInvalidTransition
	}
	j.Status = StatusCompleted
	return nil
}

func (j *Job) Fail() error {
	if j.Status != StatusCreated {
		return ErrInvalidTransition
	}
	j.Status = StatusFailed
	return nil
}
