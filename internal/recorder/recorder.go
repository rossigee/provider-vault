package recorder

import (
	"k8s.io/apimachinery/pkg/runtime"

	xpevent "github.com/crossplane/crossplane-runtime/v2/pkg/event"
)

type nopRecorder struct{}

func (r *nopRecorder) Event(_ runtime.Object, _ xpevent.Event) {}

func (r *nopRecorder) WithAnnotations(_ ...string) xpevent.Recorder { return r }

func NewNopRecorder() xpevent.Recorder {
	return &nopRecorder{}
}
