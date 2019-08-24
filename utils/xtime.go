package utils

import "time"

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error{
	t, err := time.ParseDuration(string(text))
	if err == nil {
		*d = Duration(t)
	}
	return err
}