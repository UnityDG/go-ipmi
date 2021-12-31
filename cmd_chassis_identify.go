package ipmi

// 28.5 Chassis Identify Command
// 定位
type ChassisIdentifyRequest struct {
	IdentifyInterval uint8
	ForceIdentifyOn  bool
}

type ChassisIdentifyResponse struct {
	// empty
}

func (req *ChassisIdentifyRequest) Pack() []byte {
	out := make([]byte, 2)
	packUint8(uint8(req.IdentifyInterval), out, 0)

	var force uint8 = 0
	if req.ForceIdentifyOn {
		force = 1
	}
	packUint8(force, out, 1)
	return out
}

func (req *ChassisIdentifyRequest) Command() Command {
	return CommandChassisIdentify
}

func (res *ChassisIdentifyResponse) CompletionCodes() map[uint8]string {
	return map[uint8]string{}
}

func (res *ChassisIdentifyResponse) Unpack(msg []byte) error {
	return nil
}

func (res *ChassisIdentifyResponse) Format() string {
	return ""
}

// This command causes the chassis to physically identify itself by a mechanism
// chosen by the system implementation; such as turning on blinking user-visible lights
// or emitting beeps via a speaker, LCD panel, etc.
func (c *Client) ChassisIdentify(interval uint8, force bool) (response *ChassisIdentifyResponse, err error) {
	request := &ChassisIdentifyRequest{
		IdentifyInterval: interval,
		ForceIdentifyOn:  force,
	}
	response = &ChassisIdentifyResponse{}
	err = c.Exchange(request, response)
	return
}
