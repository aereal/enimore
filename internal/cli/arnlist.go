package cli

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

type ArnList []arn.ARN

var _ flag.Value = &ArnList{}

func (a *ArnList) Set(v string) error {
	var accum []arn.ARN
	for _, x := range strings.Split(v, ",") {
		parsed, err := arn.Parse(x)
		if err != nil {
			return fmt.Errorf("cannot parse arn (%q): %w", x, err)
		}
		accum = append(accum, parsed)
	}
	*a = append(*a, accum...)
	return nil
}

func (a ArnList) String() string {
	buf := new(bytes.Buffer)
	size := len(a)
	for i, v := range a {
		buf.WriteString(v.String())
		if i == size-1 {
			break
		}
		buf.WriteByte(',')
	}
	return buf.String()
}
