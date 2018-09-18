package event

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type General struct {
	Event   string
	Payload map[string]interface{}
}

// Looking to be able to quickly split/send message event types/channels to the respective modules and marshalling everything else into a map.
func (g *General) MarshalJSON() ([]byte, error) {
	e := map[string]interface{}{}
	if g.Event != "" {
		if c, ok := e["event"]; ok {
			e["event"] = fmt.Sprintf("%s:%s", g.Event, c)
		} else {
			e["event"] = g.Event
		}
	}
	for k, v := range g.Payload {
		e[k] = v
	}
	if b, err := json.Marshal(e); err == nil {
		return b, nil
	}
	return []byte(""), nil
}

func (g *General) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if g.Payload == nil {
		g.Payload = map[string]interface{}{}
	}
	for k, v := range m {
		if k == "event" {
			if e, ok := m["event"].(string); ok {
				events := strings.Split(e, ":")
				g.Event = events[0]
				if len(events) > 1 {
					g.Payload["event"] = strings.Join(events[1:], ":")
				}
			}
		} else {
			g.Payload[k] = v
		}
	}
	return nil
}

func WrapError(err error) []byte {
	msg, merr := json.Marshal(&General{
		Event: "ERROR",
		Payload: map[string]interface{}{
			"error": err.Error(),
		},
	})
	if merr != nil {
		log.Errorf("error wrapping err.Error() %s", err.Error())
		return []byte(``)
	}
	return msg
}

func WrapValues(t string, keyvalues map[string]interface{}) []byte {
	msg, err := json.Marshal(&General{
		Event:   t,
		Payload: keyvalues,
	})
	if err != nil {
		log.Errorf("error wrapping keyvalues %v", keyvalues)
		return []byte(``)
	}
	return msg
}

func WrapValue(t string, key, value string) []byte {
	return WrapValues(t, map[string]interface{}{key: value})
}
