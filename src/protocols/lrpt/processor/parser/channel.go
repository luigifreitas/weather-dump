package parser

import (
	"fmt"
	"weather-dump/src/ccsds/frames"
	"weather-dump/src/protocols/lrpt"
	"weather-dump/src/protocols/lrpt/processor/parser/segment"
)

const maxFrameCount = 8192 * 3

// Channel struct.
type Channel struct {
	APID         uint16
	ChannelName  string
	BlockDim     int
	Invert       bool
	FinalWidth   uint32
	FileName     string
	Height       uint32
	Width        uint32
	StartTime    lrpt.Time
	EndTime      lrpt.Time
	HasData      bool
	SegmentCount uint32

	FirstSegment uint32
	LastSegment  uint32

	segments map[uint32]*segment.Data
	rollover uint32
	last     uint32
	offset   uint32
}

// NewChannel instance.
func (e *Channel) init() {
	e.HasData = true
	e.LastSegment = 0x00000000
	e.FirstSegment = 0xFFFFFFFF

	e.segments = make(map[uint32]*segment.Data)
}

func (e Channel) GetDimensions() (int, int) {
	return int(e.Width), int(e.Height)
}

// Fix the channel metadata.
func (e *Channel) Process(scft lrpt.SpacecraftParameters) {
	f := e.FirstSegment % 14
	for i := uint32(0); i < f; i++ {
		e.segments[i] = segment.NewFiller()
		e.SegmentCount++
		e.FirstSegment--
	}

	for i := e.FirstSegment; i <= e.LastSegment; i++ {
		if e.segments[i] == nil {
			e.segments[i] = segment.NewFiller()
			e.SegmentCount++
		}
	}

	e.FileName = fmt.Sprintf("%s_%s_BISMW_%s_%d", scft.Filename, scft.SignalName, e.ChannelName, e.StartTime.GetMilliseconds())
	e.Height = (e.SegmentCount + 28) * uint32(e.BlockDim) / 14
	e.Width = e.FinalWidth
}

func (e *Channel) Parse(packet frames.SpacePacketFrame) {
	if !packet.IsValid() {
		return
	}

	if new := segment.New(packet.GetData()); new.IsValid() {
		if e.last > uint32(packet.GetSequenceCount()) && e.last > 16300 {
			e.rollover += 16384
		}
		e.last = uint32(packet.GetSequenceCount())

		t := uint32(packet.GetSequenceCount()) + e.rollover + e.offset
		id := t/43*14 + uint32(new.GetMCUNumber())/14

		if !e.HasData {
			e.init()
			e.offset = t / 43
			//e.offset = (43 - (t % 43)) % 14
			//fmt.Println(e.ChannelName, t, 43-t%43)
		}

		//fmt.Println(packet.GetSequenceCount(), uint32(e.lastCount)/43*14, id, new.GetMCUNumber()/14)

		if e.LastSegment < id {
			e.LastSegment = id
			e.EndTime = new.GetDate()
		}

		if e.FirstSegment > id {
			e.FirstSegment = id
			e.StartTime = new.GetDate()
		}

		e.segments[id] = new
		e.SegmentCount++
	}
}
