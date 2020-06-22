package payload

import (
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// CreatePayload creates the in_toto link file we attach.
func CreatePayload(tr *v1beta1.TaskRun) in_toto.Link {
	l := in_toto.Link{
		Type: "_link",
	}

	l.Materials = map[string]interface{}{}
	for _, r := range tr.Spec.Resources.Inputs {
		for _, rr := range tr.Status.ResourcesResult {
			if r.Name == rr.ResourceName {
				l.Materials[rr.ResourceName] = rr
			}
		}
	}

	l.Products = map[string]interface{}{}
	for _, r := range tr.Spec.Resources.Outputs {
		for _, rr := range tr.Status.ResourcesResult {
			if r.Name == rr.ResourceName {
				l.Products[rr.ResourceName] = rr
			}
		}
	}

	l.Environment = map[string]interface{}{}
	// Add Tekton release info here
	l.Environment["tekton"] = tr.Status
	return l
}
