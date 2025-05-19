package frame

import "image"

type ImageReceiver interface {
	ChanelID() string
	Receive(data image.Image)
}
