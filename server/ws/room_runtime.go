package ws

import sessionapp "example.com/elmakina/app/session"

type roomRuntime struct {
	room *Room
}

func (rt *roomRuntime) dispatchInteraction(interaction *sessionapp.InteractionState) error {
	envelope, err := BuildPromptEnvelope(interaction)
	if err != nil {
		return err
	}
	for _, playerIndex := range interaction.Recipients.Players {
		if _, err := rt.room.SendToPlayerIfConnected(playerIndex, envelope); err != nil {
			return err
		}
	}
	return nil
}
