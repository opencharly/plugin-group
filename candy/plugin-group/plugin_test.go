package groupkind

// Unit gates for the member-tree migration (Cutover C task 0): the OpLoad reply reconstructs the
// ONE ordered spec.Deploy.Member tree from the host-threaded env — every entry a deploy-level PEER
// (Alongside) — with Target forced empty. No dual Children/Members map is ever constructed.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

func invoke(t *testing.T, req *pb.InvokeRequest) spec.Deploy {
	t.Helper()
	reply, err := provider{}.Invoke(t.Context(), req)
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	var dep spec.Deploy
	if err := json.Unmarshal(reply.ResultJson, &dep); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	return dep
}

func TestInvokeReconstructsMemberTree(t *testing.T) {
	env, err := json.Marshal(spec.StructuralKindLoadEnv{Members: map[string]*spec.Deploy{
		"pg":    {},
		"cache": {},
	}})
	if err != nil {
		t.Fatalf("marshal env: %v", err)
	}
	dep := invoke(t, &pb.InvokeRequest{
		Op:         sdk.OpLoad,
		ParamsJson: []byte(`{"disposable": true}`),
		EnvJson:    env,
	})
	if dep.Target != "" {
		t.Errorf("Target = %q, want empty (a group is targetless)", dep.Target)
	}
	if !dep.HasMembers() || len(dep.Member) != 2 {
		t.Fatalf("HasMembers/len(Member) = %v/%d, want true/2", dep.HasMembers(), len(dep.Member))
	}
	// Deterministic canonical order (the env map loses authored order): sorted names.
	if dep.Member[0].Name != "cache" || dep.Member[1].Name != "pg" {
		t.Errorf("member order = [%s, %s], want [cache pg]", dep.Member[0].Name, dep.Member[1].Name)
	}
	for _, m := range dep.Member {
		if !m.Alongside() || m.Position != spec.PositionDeployLevel {
			t.Errorf("member %q: position %q Alongside=%v, want deploy-level/true (a group's members are PEERS)", m.Name, m.Position, m.Alongside())
		}
		if m.Node == nil {
			t.Errorf("member %q: nil Node", m.Name)
		}
	}
	if dep.InSubstrateMembers() != nil {
		t.Errorf("InSubstrateMembers = %v, want none (a group has no venue to deploy into)", dep.InSubstrateMembers())
	}
	if got := dep.MemberByName("pg"); got == nil || got.Name != "pg" {
		t.Errorf("MemberByName(\"pg\") = %v, want the pg entry", got)
	}
	if got := dep.MemberByName("absent"); got != nil {
		t.Errorf("MemberByName(\"absent\") = %v, want nil", got)
	}
}

func TestInvokeNoEnvYieldsEmptyTree(t *testing.T) {
	dep := invoke(t, &pb.InvokeRequest{Op: sdk.OpLoad, ParamsJson: []byte(`{}`)})
	if dep.HasMembers() {
		t.Errorf("HasMembers = true, want false (no authored members)")
	}
	if dep.Target != "" {
		t.Errorf("Target = %q, want empty", dep.Target)
	}
}

func TestInvokeRejectsUnsupportedOp(t *testing.T) {
	_, err := provider{}.Invoke(t.Context(), &pb.InvokeRequest{Op: "op:other"})
	if err == nil || !strings.Contains(err.Error(), "unsupported op") {
		t.Errorf("err = %v, want unsupported-op failure", err)
	}
}
