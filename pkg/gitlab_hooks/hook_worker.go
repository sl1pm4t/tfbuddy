package gitlab_hooks

import (
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/zapier/tfbuddy/pkg/runstream"
	"github.com/zapier/tfbuddy/pkg/tfc_api"
	"github.com/zapier/tfbuddy/pkg/tfc_trigger"
	"github.com/zapier/tfbuddy/pkg/vcs"
)

type GitlabEventWorker struct {
	tfc             tfc_api.ApiClient
	gl              vcs.GitClient
	runstream       runstream.StreamClient
	triggerCreation TriggerCreationFunc
}

func NewGitlabEventWorker(h *GitlabHooksHandler, js nats.JetStreamContext) *GitlabEventWorker {
	w := &GitlabEventWorker{
		tfc:             h.tfc,
		gl:              h.gl,
		runstream:       h.runstream,
		triggerCreation: tfc_trigger.NewTFCTrigger,
	}

	_, err := h.mrStream.QueueSubscribe("gitlab_mr_event_worker", w.processMREventStreamMsg)
	if err != nil {
		log.Error().Err(err).Msg("could not subscribe to hook stream")
	}

	_, err = h.notesStream.QueueSubscribe("gitlab_note_event_worker", w.processNoteEventStreamMsg)
	if err != nil {
		log.Error().Err(err).Msg("could not subscribe to hook stream")
	}

	return w
}

func (w *GitlabEventWorker) processNoteEventStreamMsg(msg *NoteEventMsg) error {
	log.Debug().Caller().Msg("got gitlab NoteEvent from hook stream")
	_, err := w.processNoteEvent(msg)
	return err
}

func (w *GitlabEventWorker) processMREventStreamMsg(msg *MergeRequestEventMsg) error {
	log.Debug().Caller().Msg("got gitlab MergeRequestEventMsg from hook stream")
	_, err := w.processMergeRequestEvent(msg)
	return err
}
