package docker

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainer_Run(t *testing.T) {
	d, err := New("golang", "echo Hello World")
	assert.Equal(t, err, nil)
	err = d.Run(context.Background())
	assert.Equal(t, err, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func TestContainer_Logs(t *testing.T) {
	d, err := New("golang", "echo Hello World")
	assert.Equal(t, err, nil)
	err = d.Run(context.Background())
	assert.Equal(t, err, nil)
	d.logging(context.Background())
	msg := <-d.Logs()
	assert.Equal(t, msg, "Hello World")
}
