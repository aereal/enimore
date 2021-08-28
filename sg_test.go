package enimore

import (
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/google/go-cmp/cmp"
)

func TestSecurityGroupAssociation(t *testing.T) {
	serviceARN1 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-1/my-service"
	serviceARN2 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-2/my-service-2"
	securityGroupIDs1 := []string{"sg-1234567890", "sg-987654321"}
	securityGroupIDs2 := []string{"sg-8765432109", "sg-7654321098"}

	// initial
	a := &securityGroupAssociation{sgID2Resource: map[string]arn.ARN{}}
	t.Run("initial", func(t *testing.T) {
		if a.hasAny() {
			t.Errorf("hasAny(): must return false")
		}
		if diff := diffStringSlice([]string{}, a.securityGroupIDs()); diff != "" {
			t.Errorf("securityGroupIDs() (-want, +got):\n%s", diff)
		}
		if _, ok := a.get(securityGroupIDs1[0]); ok {
			t.Errorf("get(%s): must return false", securityGroupIDs1[0])
		}
		if _, ok := a.get(securityGroupIDs2[0]); ok {
			t.Errorf("get(%s): must return false", securityGroupIDs2[0])
		}
	})

	// add serviceARN1
	a.add(mustParseARN(serviceARN1), securityGroupIDs1...)
	t.Run("added serviceARN1", func(t *testing.T) {
		if !a.hasAny() {
			t.Errorf("hasAny(): must return true")
		}
		if diff := diffStringSlice(securityGroupIDs1, a.securityGroupIDs()); diff != "" {
			t.Errorf("securityGroupIDs() (-want, +got):\n%s", diff)
		}
		if got, ok := a.get(securityGroupIDs1[0]); true {
			if !ok {
				t.Errorf("get(%s): must return ok", securityGroupIDs1[0])
			}
			if got.String() != serviceARN1 {
				t.Errorf("get(%s) arn (-want, +got):\n%s", securityGroupIDs1[0], cmp.Diff(serviceARN1, got.String()))
			}
		}
		if _, ok := a.get(securityGroupIDs2[0]); ok {
			t.Errorf("get(%s): must return false", securityGroupIDs2[0])
		}
	})
	a.add(mustParseARN(serviceARN2), securityGroupIDs2...)
	t.Run("added serviceARN1", func(t *testing.T) {
		if !a.hasAny() {
			t.Errorf("hasAny(): must return true")
		}
		if diff := diffStringSlice(append(securityGroupIDs1, securityGroupIDs2...), a.securityGroupIDs()); diff != "" {
			t.Errorf("securityGroupIDs() (-want, +got):\n%s", diff)
		}
		if got, ok := a.get(securityGroupIDs1[0]); true {
			if !ok {
				t.Errorf("get(%s): must return ok", securityGroupIDs1[0])
			}
			if got.String() != serviceARN1 {
				t.Errorf("get(%s) arn (-want, +got):\n%s", securityGroupIDs1[0], cmp.Diff(serviceARN1, got.String()))
			}
		}
		if got, ok := a.get(securityGroupIDs2[0]); true {
			if !ok {
				t.Errorf("get(%s): must return ok", securityGroupIDs2[0])
			}
			if got.String() != serviceARN2 {
				t.Errorf("get(%s) arn (-want, +got):\n%s", securityGroupIDs2[0], cmp.Diff(serviceARN2, got.String()))
			}
		}
	})
}

func diffStringSlice(want, got []string) string {
	return cmp.Diff(want, got, cmp.Transformer("sorted", func(in []string) []string {
		out := append([]string(nil), in...)
		sort.Strings(out)
		return out
	}))
}
