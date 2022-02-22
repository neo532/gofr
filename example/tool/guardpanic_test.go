package tool

import (
	"context"
	"fmt"
	"testing"

	"github.com/neo532/gofr/tool"
)

func TestGuardpanic(t *testing.T) {
	c := context.Background()
	fn := func() {
		fmt.Println(fmt.Sprintf("%s\t:Biz run!", t.Name()))
	}
	tool.Run(c, fn, 1, logger.CErr)
}
