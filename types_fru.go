package ipmi

import (
	"bytes"
	"fmt"
	"time"
)

// 38. Accessing FRU Devices
//
// FRU devices can located in three different types of location.
type FRULocation string

const (
	FRULocation_IPMB           FRULocation = "directly on IPMB"
	FRULocation_PrivateBus     FRULocation = "on private bus"
	FRULocation_MgmtController FRULocation = "on mangement controller"
)

const (
	FRUFormatVersion     uint8 = 0x01
	FRUAreaFieldsEndMark uint8 = 0xc1
	FRUCommonHeaderSize  uint8 = 8
)

type FRU struct {
	deviceID         uint8
	deviceName       string
	deviceNotPresent bool

	CommandHeader   *FRUCommonHeader
	InternalUseArea *FRUInternalUseArea
	ChassisInfoArea *FRUChassisInfoArea
	BoardInfoArea   *FRUBoardInfoArea
	ProductInfoArea *FRUProductInfoArea
	MultiRecords    []*FRUMultiRecord
}

func (fru *FRU) Present() bool {
	return !fru.deviceNotPresent
}

func (fru *FRU) DeviceName() string {
	return fru.deviceName
}

func (fru *FRU) DeviceID() uint8 {
	return fru.deviceID
}

func (fru *FRU) String() string {
	var buf = new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("FRU Device Description : %s (ID %d)\n", fru.deviceName, fru.deviceID))
	if !fru.Present() {
		buf.WriteString("  Device not present\n")
		return buf.String()
	}

	if fru.ChassisInfoArea != nil {
		buf.WriteString(fmt.Sprintf("  Chassis Type         : %s\n", fru.ChassisInfoArea.ChassisType.String()))
		buf.WriteString(fmt.Sprintf("  Chassis Part Number  : %s\n", fru.ChassisInfoArea.PartNumber))
		buf.WriteString(fmt.Sprintf("  Chassis Serial Number: %s\n", fru.ChassisInfoArea.SerialNumber))
		for _, v := range fru.ChassisInfoArea.Custom {
			buf.WriteString(fmt.Sprintf("  Chassis Extra        : %s\n", v))
		}
	}

	if fru.BoardInfoArea != nil {
		buf.WriteString(fmt.Sprintf("  Board Mfg Date       : %s\n", fru.BoardInfoArea.MfgDateTime.String()))
		buf.WriteString(fmt.Sprintf("  Board Mfg            : %s\n", fru.BoardInfoArea.Manufacturer))
		buf.WriteString(fmt.Sprintf("  Board Product        : %s\n", fru.BoardInfoArea.ProductName))
		buf.WriteString(fmt.Sprintf("  Board Serial         : %s\n", fru.BoardInfoArea.SerialNumber))
		buf.WriteString(fmt.Sprintf("  Board Part Number    : %s\n", fru.BoardInfoArea.PartNumber))
		for _, v := range fru.BoardInfoArea.Custom {
			buf.WriteString(fmt.Sprintf("  Board Extra          : %s\n", v))
		}
	}

	if fru.ProductInfoArea != nil {
		buf.WriteString(fmt.Sprintf("  Product Mfg          : %s\n", fru.ProductInfoArea.Manufacturer))
		buf.WriteString(fmt.Sprintf("  Product Name         : %s\n", fru.ProductInfoArea.Name))
		buf.WriteString(fmt.Sprintf("  Product Part Number  : %s\n", fru.ProductInfoArea.PartModel))
		buf.WriteString(fmt.Sprintf("  Product Serial       : %s\n", fru.ProductInfoArea.SerialNumber))
		for _, v := range fru.ProductInfoArea.Custom {
			buf.WriteString(fmt.Sprintf("  Product Extra        : %s\n", v))
		}
	}
	for _, multiRecord := range fru.MultiRecords {
		buf.WriteString(fmt.Sprintf("  Multi Record         : %s\n", multiRecord.RecordType.String()))
	}
	return buf.String()
}

// The offset unit in wire is in multiples of 8 bytes, offset value 0x0 indicates that this area is not present.
// 8. Common Header Format
// 17. FRU Information Layout
type FRUCommonHeader struct {
	FormatVersion        uint8
	InternalOffset8B     uint8
	ChassisOffset8B      uint8
	BoardOffset8B        uint8
	ProductOffset8B      uint8
	MultiRecordsOffset8B uint8
	Checksum             uint8
}

func (s *FRUCommonHeader) Pack() []byte {
	out := make([]byte, 8)
	packUint8(s.FormatVersion, out, 0)
	packUint8(s.InternalOffset8B, out, 1)
	packUint8(s.ChassisOffset8B, out, 2)
	packUint8(s.BoardOffset8B, out, 3)
	packUint8(s.ProductOffset8B, out, 4)
	packUint8(s.MultiRecordsOffset8B, out, 5)
	// a pad byte at index 6
	packUint8(s.Checksum, out, 7)

	return out
}

func (s *FRUCommonHeader) Unpack(msg []byte) error {
	if len(msg) < 8 {
		return ErrUnpackedDataTooShort
	}
	s.FormatVersion, _, _ = unpackUint8(msg, 0)
	s.InternalOffset8B, _, _ = unpackUint8(msg, 1)
	s.ChassisOffset8B, _, _ = unpackUint8(msg, 2)
	s.BoardOffset8B, _, _ = unpackUint8(msg, 3)
	s.ProductOffset8B, _, _ = unpackUint8(msg, 4)
	s.MultiRecordsOffset8B, _, _ = unpackUint8(msg, 5)
	s.Checksum, _, _ = unpackUint8(msg, 7)
	return nil
}

func (s *FRUCommonHeader) Valid() bool {
	var checksumFn = func(msg []byte, start int, end int) uint8 {
		c := 0
		for i := start; i < end; i++ {
			c = (c + int(msg[i])) % 256
		}
		return -uint8(c)
	}
	msg := s.Pack()
	return s.Checksum == checksumFn(msg, 0, 6)
}

func (s *FRUCommonHeader) String() string {
	return fmt.Sprintf(`Version            : %#02x
Offset Internal    : %#02x
Offset Chassis     : %#02x
Offset Board       : %#02x
Offset Product     : %#02x
Offset MultiRecord : %#02x`,
		s.FormatVersion,
		s.InternalOffset8B*8,
		s.ChassisOffset8B*8,
		s.BoardOffset8B*8,
		s.ProductOffset8B*8,
		s.MultiRecordsOffset8B*8,
	)
}

type FRUInternalUseArea struct {
	FormatVersion uint8
	Data          []byte
}

type FRUChassisInfoArea struct {
	FormatVersion          uint8
	Length8B               uint8
	ChassisType            ChassisType
	PartNumberTypeLength   TypeLength
	PartNumber             []byte
	SerialNumberTypeLength TypeLength
	SerialNumber           []byte
	Custom                 [][]byte
	Unused                 []byte
	Checksum               uint8
}

func (fruChassis *FRUChassisInfoArea) Unpack(msg []byte) error {
	if len(msg) < 2 {
		return ErrUnpackedDataTooShort
	}
	// Chassis Info Area Length (in multiples of 8 bytes)
	if len(msg) < int(msg[1])*8 {
		return ErrUnpackedDataTooShort
	}

	fruChassis.FormatVersion = msg[0]
	fruChassis.Length8B = msg[1]
	fruChassis.ChassisType = ChassisType(msg[2])

	var offset uint16 = 3
	var err error

	offset, fruChassis.PartNumberTypeLength, fruChassis.PartNumber, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru chassis part number field failed, err: %s", err)
	}

	offset, fruChassis.SerialNumberTypeLength, fruChassis.SerialNumber, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru chassis serial number field failed, err: %s", err)
	}

	fruChassis.Custom, fruChassis.Unused, fruChassis.Checksum, err = getFRUCustomUnusedChecksumFields(msg, offset)
	if err != nil {
		return fmt.Errorf("getFRUCustomUnusedChecksumFields failed, err: %s", err)
	}

	return nil
}

type ChassisType uint8

func (chassisType ChassisType) String() string {
	// SMBIOS Specification: Table 17 - System Enclosure or Chassis Types
	var chassisTypeMaps = map[ChassisType]string{
		0x00: "Unspecified",
		0x01: "Other",
		0x02: "Unknown",
		0x03: "Desktop",
		0x04: "Low Profile Desktop",
		0x05: "Pizza Box",
		0x06: "Mini Tower",
		0x07: "Tower",
		0x08: "Portable",
		0x09: "Laptop",
		0x0a: "Notebook",
		0x0b: "Hand Held",
		0x0c: "Docking Station",
		0x0d: "All in One",
		0x0e: "Sub Notebook",
		0x0f: "Space-saving",
		0x10: "Lunch Box",
		0x11: "Main Server Chassis",
		0x12: "Expansion Chassis",
		0x13: "SubChassis",
		0x14: "Bus Expansion Chassis",
		0x15: "Peripheral Chassis",
		0x16: "RAID Chassis",
		0x17: "Rack Mount Chassis",
		0x18: "Sealed-case PC",
		0x19: "Multi-system chassis",
		0x1a: "Compact PCI",
		0x1b: "Advanced TCA",
		0x1c: "Blade",
		0x1d: "Blade Enclosure",
		0x1e: "Tablet",
		0x1f: "Convertible",
		0x20: "Detachable",
		0x21: "IoT Gateway",
		0x22: "Embedded PC",
		0x23: "Mini PC",
		0x24: "Stick PC",
	}

	s, ok := chassisTypeMaps[chassisType]
	if ok {
		return s
	}

	return ""
}

type ChassisState uint8

func (chassisState ChassisState) String() string {
	// SMBIOS Specification: Table 18 - System Enclosure or Chassis States
	var chassisStateMap = map[ChassisState]string{
		0x01: "Other",
		0x02: "Unknown",
		0x03: "Safe",
		0x04: "Warning",
		0x05: "Critical",
		0x06: "Non-recoverable",
	}
	if s, ok := chassisStateMap[chassisState]; ok {
		return s
	}
	return ""
}

type ChassisSecurityStatus uint8

func (chassisSecurityStatus ChassisSecurityStatus) String() string {
	// SMBIOS Specification: // Table 19 - System Enclosure or Chassis Security Status field
	var chassisSecurityStatusMap = map[ChassisSecurityStatus]string{
		0x01: "Other",
		0x02: "Unknown",
		0x03: "None",
		0x04: "External interface locked out",
		0x05: "External interface enabled",
	}

	if s, ok := chassisSecurityStatusMap[chassisSecurityStatus]; ok {
		return s
	}
	return ""
}

// fru: 11. Board Info Area Format
type FRUBoardInfoArea struct {
	FormatVersion          uint8
	Length8B               uint8
	LanguageCode           uint8
	MfgDateTime            time.Time
	ManufacturerTypeLength TypeLength
	Manufacturer           []byte
	ProductNameTypeLength  TypeLength
	ProductName            []byte
	SerialNumberTypeLength TypeLength
	SerialNumber           []byte
	PartNumberTypeLength   TypeLength
	PartNumber             []byte
	FRUFileIDTypeLength    TypeLength
	FRUFileID              []byte
	Custom                 [][]byte
	Unused                 []byte
	Checksum               uint8
}

func (fruBoard *FRUBoardInfoArea) Unpack(msg []byte) error {
	if len(msg) < 2 {
		return ErrUnpackedDataTooShort
	}
	// Board Area Length (in multiples of 8 bytes)
	if len(msg) < int(msg[1])*8 {
		return ErrUnpackedDataTooShort
	}

	fruBoard.FormatVersion = msg[0]
	fruBoard.Length8B = msg[1]
	fruBoard.LanguageCode = msg[2]

	m, _, _ := unpackUint24L(msg, 3) // Number of minutes from 0:00 hrs 1/1/96.
	const secsFrom1970To1996 uint32 = 820454400
	fruBoard.MfgDateTime = parseTimestamp(secsFrom1970To1996 + m*60)

	var offset uint16 = 6
	var err error

	offset, fruBoard.ManufacturerTypeLength, fruBoard.Manufacturer, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru board manufacturer field failed, err: %s", err)
	}

	offset, fruBoard.ProductNameTypeLength, fruBoard.ProductName, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru board product name field failed, err: %s", err)
	}

	offset, fruBoard.SerialNumberTypeLength, fruBoard.SerialNumber, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru board serial number field failed, err: %s", err)
	}

	offset, fruBoard.PartNumberTypeLength, fruBoard.PartNumber, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru board part number field failed, err: %s", err)
	}

	offset, fruBoard.FRUFileIDTypeLength, fruBoard.FRUFileID, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru board file id field failed, err: %s", err)
	}

	fruBoard.Custom, fruBoard.Unused, fruBoard.Checksum, err = getFRUCustomUnusedChecksumFields(msg, offset)
	if err != nil {
		return fmt.Errorf("getFRUCustomUnusedChecksumFields failed, err: %s", err)
	}

	return nil
}

type BoardType uint8

func (boardType BoardType) String() string {
	var boardTypeMap = map[BoardType]string{
		0x01: "Unknown",
		0x02: "Other",
		0x03: "Server Blade",
		0x04: "Connectivity Switch",
		0x05: "System Management Module",
		0x06: "Processor Module",
		0x07: "I/O Module",
		0x08: "Memory Module",
		0x09: "Daughter board",
		0x0a: "Motherboard",
		0x0b: "Processor/Memory Module",
		0x0c: "Processor/IO Module",
		0x0d: "Interconnect board",
	}

	if s, ok := boardTypeMap[boardType]; ok {
		return s
	}
	return ""
}

type FRUProductInfoArea struct {
	FormatVersion          uint8
	Length8B               uint8
	LanguageCode           uint8
	ManufacturerTypeLength TypeLength
	Manufacturer           []byte
	NameTypeLength         TypeLength
	Name                   []byte
	PartModelTypeLength    TypeLength
	PartModel              []byte
	VersionTypeLength      TypeLength
	Version                []byte
	SerialNumberTypeLength TypeLength
	SerialNumber           []byte
	AssetTagTypeLength     TypeLength
	AssetTag               []byte
	FRUFileIDTypeLength    TypeLength
	FRUFileID              []byte
	Custom                 [][]byte
	Unused                 []byte
	Checksum               uint8
}

func (fruProduct *FRUProductInfoArea) Unpack(msg []byte) error {
	if len(msg) < 2 {
		return ErrUnpackedDataTooShort
	}
	// Product Area Length (in multiples of 8 bytes)
	if len(msg) < int(msg[1])*8 {
		return ErrUnpackedDataTooShort
	}

	fruProduct.FormatVersion = msg[0]
	fruProduct.Length8B = msg[1]
	fruProduct.LanguageCode = msg[2]

	var offset uint16 = 3
	var err error

	offset, fruProduct.ManufacturerTypeLength, fruProduct.Manufacturer, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product manufacturer field failed, err: %s", err)
	}

	offset, fruProduct.NameTypeLength, fruProduct.Name, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product name field failed, err: %s", err)
	}

	offset, fruProduct.PartModelTypeLength, fruProduct.PartModel, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product part model field failed, err: %s", err)
	}

	offset, fruProduct.VersionTypeLength, fruProduct.Version, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product version field failed, err: %s", err)
	}

	offset, fruProduct.SerialNumberTypeLength, fruProduct.SerialNumber, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product serial number field failed, err: %s", err)
	}

	offset, fruProduct.AssetTagTypeLength, fruProduct.AssetTag, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product asset tag field failed, err: %s", err)
	}

	offset, fruProduct.FRUFileIDTypeLength, fruProduct.FRUFileID, err = getFRUTypeLengthField(msg, offset)
	if err != nil {
		return fmt.Errorf("get fru product file id field failed, err: %s", err)
	}

	fruProduct.Custom, fruProduct.Unused, fruProduct.Checksum, err = getFRUCustomUnusedChecksumFields(msg, offset)
	if err != nil {
		return fmt.Errorf("getFRUCustomUnusedChecksumFields failed, err: %s", err)
	}

	return nil
}

type FRUMultiRecord struct {
	RecordType FRURecordType // used to identify the information contained in the record

	EndOfList bool // indicates if this record is the last record in the MultiRecord area

	// Record Format version (=2h unless otherwise specified)
	// This field is used to identify the revision level of information stored in this area.
	// This number will start at zero for each new area. If changes need to be made to the record,
	// e.g. fields added/removed, the version number will be increased to reflect the change.
	FormatVersion uint8

	// RecordLength indicates the number of bytes of data in the record. This byte can also be used to find the
	// next area in the list. If the "End of List" bit is zero, the length can be added the starting offset of the current
	// Record Data to get the offset of the next Record Header. This field allows for 0 to 255 bytes of data for
	// each record.
	RecordLength uint8

	RecordChecksum uint8
	HeaderChecksum uint8

	RecordData []byte
}

func (fruMultiRecord *FRUMultiRecord) Unpack(msg []byte) error {
	if len(msg) < 3 {
		return ErrUnpackedDataTooShort
	}
	// RecordLength
	if len(msg) < int(msg[2]) {
		return ErrUnpackedDataTooShort
	}

	fruMultiRecord.RecordType = FRURecordType(msg[0])

	b1 := msg[1]
	fruMultiRecord.EndOfList = isBit7Set(b1)
	fruMultiRecord.FormatVersion = b1 & 0x0f

	fruMultiRecord.RecordLength = msg[2]
	fruMultiRecord.RecordChecksum = msg[3]
	fruMultiRecord.HeaderChecksum = msg[4]

	dataLen := int(fruMultiRecord.RecordLength) - 5
	fruMultiRecord.RecordData, _, _ = unpackBytes(msg, 5, dataLen)

	return nil
}

type FRURecordType uint8

func (t FRURecordType) String() string {
	// fru: Table 16-2, MultiRecord Area Record Types
	m := map[FRURecordType]string{
		0x00: "Power Supply",
		0x01: "DC Output",
		0x02: "DC Load",
		0x03: "Management Access",
		0x04: "Base Compatibility",
		0x05: "Extended Compatibility",
		0x06: "ASF Fixed SMBus Device",   // see [ASF_2.0] for definition
		0x07: "ASF Legacy-Device Alerts", // see [ASF_2.0] for definition
		0x08: "ASF Remote Control",       // see [ASF_2.0] for definition
		0x09: "Extended DC Output",
		0x0a: "Extended DC Load",
		// 0x0b-0x0f reserved for definition by working group, Refer to specifications from the NVM Express™ working group (www.nvmexpress.org)
		// 0x10-0xbf reserved
		// 0xc0-0xff OEM Record Types
	}
	s, ok := m[t]
	if ok {
		return s
	}
	return ""
}

// fru: 18.1 Power Supply Information (Record Type 0x00)
type FRURecordTypePowerSupply struct {
	// This field allows for Power Supplies with capacities from 0 to 4095 watts.
	OverallCapacity uint16
	// The highest instantaneous VA value that this supply draws during operation (other than during Inrush). In integer units. FFFFh if not specified.
	PeakVA uint16
	// Maximum inrush of current, in Amps, into the power supply. FFh if not specified.
	InrushCurrent uint8 // 涌入电流
	// Number of milliseconds before power supply loading enters non-startup operating range. Set to 0 if no inrush current specified.
	InrushIntervalMilliSecond uint8
	// This specifies the low end of acceptable voltage into the power supply. The units are 10mV.
	LowEndInputVoltageRange1 uint16
	// This specifies the high end of acceptable voltage into the power supply. The units are 10mV.
	HighEndInputVoltageRange1 uint16
	// This specifies the low end of acceptable voltage into the power supply. This field would be used if the power supply did not support autoswitch. Range 1 would define the 110V range, while range 2 would be used for 220V. The units are 10mV.
	LowEndInputVoltageRange2 uint16
	// This specifies the high end of acceptable voltage into the power supply. This field would be used if the power supply did not support autoswitch. Range 1 would define the 110V range, while range 2 would be used for 220V. The units are 10mV.
	HighEndInputVoltageRange2 uint16
	// This specifies the low end of acceptable frequency range into the power supply. Use 00h if supply accepts a DC input.
	LowEndInputFrequencyRange uint8
	// This specifies the high end of acceptable frequency range into the power supply. Use 00h for both Low End and High End frequency range if supply only takes a DC input.
	HighEndInputFrequencyRange uint8
	// Minimum number of milliseconds the power supply can hold up POWERGOOD (and maintain valid DC output) after input power is lost.
	InputDropoutToleranceMilliSecond uint8

	HotSwapSuppot         bool
	Autoswitch            bool
	PowerFactorCorrection bool
	PredictiveFailSupport bool

	// the number of seconds peak wattage can be sustained (0-15 seconds)
	PeakWattageHoldupSecond uint8
	// the peak wattage the power supply can produce during this time period
	PeakCapacity uint16

	CombinedWattageVoltage1 uint8 // bit 7:4 - Voltage 1
	CombinedWattageVoltage2 uint8 // bit 3:0 - Voltage 2
	// 0000b (0) 12V
	// 0001b (1) -12V
	// 0010b (2) 5V
	// 0011b (3) 3.3V

	TotalCombinedWattage uint16

	// This field serves two purposes.
	// It clarifies what type of predictive fail the power supply supports
	// (pass/fail signal or the tachometer output of the power supply fan)
	// and indicates the predictive failing point for tach outputs.
	// This field should be written as zero and ignored if the
	// predictive failure pin of the power supply is not supported.
	//
	// 	0x00 Predictive fail pin indicates pass/fail
	//  0x01 - 0xFF Lower threshold to indicate predictive failure (Rotations per second)
	PredictiveFailTachometerLowerThreshold uint8 // RPS
}

// FRU: 18.2 DC Output (Record Type 0x01)
type FRURecordTypeDCOutput struct {
	//  if the power supply provides this output even when the power supply is switched off.
	OutputWhenOff bool

	OutputNumber uint8

	// Expected voltage from the power supply. Value is a signed short given in 10 millivolt increments.
	// 额定电压 毫-伏特
	NominalVoltage10mV int16

	MaxNegativeVoltage10mV int16

	MaxPositiveVoltage10mV int16

	RippleNoise1mV uint16

	// 毫-安培
	MinCurrentDraw1mA uint16

	MaxCurrentDraw1mA uint16
}

func (output *FRURecordTypeDCOutput) Unpack(msg []byte) error {
	if len(msg) < 12 {
		return ErrUnpackedDataTooShort
	}
	b, _, _ := unpackUint8(msg, 0)
	output.OutputWhenOff = isBit7Set(b)
	output.OutputNumber = b & 0x0f

	b1, _, _ := unpackUint16L(msg, 1)
	output.NominalVoltage10mV = int16(b1)

	b3, _, _ := unpackUint16L(msg, 3)
	output.MaxNegativeVoltage10mV = int16(b3)

	b5, _, _ := unpackUint16L(msg, 5)
	output.MaxPositiveVoltage10mV = int16(b5)

	output.RippleNoise1mV, _, _ = unpackUint16L(msg, 7)
	output.MinCurrentDraw1mA, _, _ = unpackUint16L(msg, 9)
	output.MaxCurrentDraw1mA, _, _ = unpackUint16L(msg, 11)

	return nil
}

// FRU: 18.2a Extended DC Output (Record Type 0x09)
type FRURecordTypeExtenedDCOutput struct {
	//  if the power supply provides this output even when the power supply is switched off.
	OutputWhenOff bool

	// This record can be used to support power supplies with outputs that exceed 65.535 Amps.
	// 0b = 10 mA
	// 1b = 100 mA
	CurrentUnits100 bool

	OutputNumber uint8

	// Expected voltage from the power supply. Value is a signed short given in 10 millivolt increments.
	// 毫-伏特
	NominalVoltage10mV int16

	MaxNegativeVoltage10mV int16

	MaxPositiveVoltage10mV int16

	RippleNoise uint16

	// The unit is determined by CurrentUnits100 field.
	MinCurrentDraw uint16
	MaxCurrentDraw uint16
}

func (output *FRURecordTypeExtenedDCOutput) Unpack(msg []byte) error {
	if len(msg) < 12 {
		return ErrUnpackedDataTooShort
	}
	b, _, _ := unpackUint8(msg, 0)
	output.OutputWhenOff = isBit7Set(b)
	output.CurrentUnits100 = isBit4Set(b)
	output.OutputNumber = b & 0x0f

	b1, _, _ := unpackUint16L(msg, 1)
	output.NominalVoltage10mV = int16(b1)

	b3, _, _ := unpackUint16L(msg, 3)
	output.MaxNegativeVoltage10mV = int16(b3)

	b5, _, _ := unpackUint16L(msg, 5)
	output.MaxPositiveVoltage10mV = int16(b5)

	output.RippleNoise, _, _ = unpackUint16L(msg, 7)
	output.MinCurrentDraw, _, _ = unpackUint16L(msg, 9)
	output.MaxCurrentDraw, _, _ = unpackUint16L(msg, 11)

	return nil
}

// FRU: 18.3 DC Load (Record Type 0x02)
type FRURecordTypeDCLoad struct {
	OutputNumber            uint8
	NominalVoltage10mV      int16
	MinTolerableVoltage10mV int16
	MaxTolerableVoltage10mV int16
	RippleNoise1mV          uint16
	MinCurrentLoad1mA       uint16
	MaxCurrentLoad1mA       uint16
}

func (output *FRURecordTypeDCLoad) Unpack(msg []byte) error {
	if len(msg) < 12 {
		return ErrUnpackedDataTooShort
	}
	b, _, _ := unpackUint8(msg, 0)
	output.OutputNumber = b & 0x0f

	b1, _, _ := unpackUint16L(msg, 1)
	output.NominalVoltage10mV = int16(b1)

	b3, _, _ := unpackUint16L(msg, 3)
	output.MinTolerableVoltage10mV = int16(b3)

	b5, _, _ := unpackUint16L(msg, 5)
	output.MaxTolerableVoltage10mV = int16(b5)

	output.RippleNoise1mV, _, _ = unpackUint16L(msg, 7)
	output.MinCurrentLoad1mA, _, _ = unpackUint16L(msg, 9)
	output.MaxCurrentLoad1mA, _, _ = unpackUint16L(msg, 11)

	return nil
}

// FRU: 18.3a Extended DC Load (Record Type 0x0A)
type FRURecordTypeExtendedDCLoad struct {
	IsCurrrentUnit100mA bool // current units: true = 100 mA , false = 10 mA
	OutputNumber        uint8
	NominalVoltage10mV  int16
	MinVoltage10mV      int16
	MaxVoltage10mV      int16
	RippleNoise1mV      int16
	MinCurrentLoad      uint16 // units is determined by IsCurrentUnit100mA field
	MaxCurrentLoad      uint16 // units is determined by IsCurrentUnit100mA field
}

func (f *FRURecordTypeExtendedDCLoad) Unpack(msg []byte) error {
	if len(msg) < 13 {
		return ErrUnpackedDataTooShort
	}
	f.IsCurrrentUnit100mA = isBit7Set(msg[0])
	f.OutputNumber = msg[0] & 0x0f

	b1, _, _ := unpackUint16L(msg, 1)
	f.NominalVoltage10mV = int16(b1)

	b3, _, _ := unpackUint16L(msg, 3)
	f.MinVoltage10mV = int16(b3)

	b5, _, _ := unpackUint16L(msg, 5)
	f.MaxVoltage10mV = int16(b5)

	b7, _, _ := unpackUint16L(msg, 7)
	f.RippleNoise1mV = int16(b7)

	f.MinCurrentLoad, _, _ = unpackUint16L(msg, 9)
	f.MaxCurrentLoad, _, _ = unpackUint16L(msg, 11)

	return nil
}

type ManagementAccessSubRecordType uint8

func (t ManagementAccessSubRecordType) String() string {
	m := map[ManagementAccessSubRecordType]string{
		// SystemMgmtURL []byte
		// // A name to identify the system that contains this FRU. (same as DMI
		// // DMTF|General Information|001 - System Name)
		// SystemName []byte
		// // The IP network address of the system that contains this FRU. Can be either the IP
		// // address or the host name + domain name (eg. finance.sc.hp.com)
		// SystemPingAddr []byte
		// // The Internet Uniform Resource Locator string that can be used through a World
		// // Wide Web browser to obtain management information about this FRU. (same as DMI
		// // DMTF|Field Replaceable Unit|002 - FRU Internet Uniform Resource Locator)
		// ComponentMgmtURL []byte
		// // A clear description of this FRU. (same asDMI "DMTF|Field Replaceable Unit|002 - Description")
		// ComponentName []byte
		// // The IP network address of this FRU. Can be either the IP address or the host name
		// // + domain name (e.g. critter.sc.hp.com).
		// ComponentPingAddr []byte
		// // This is a copy of the system GUID from [SMBIOS]
		// SystemUniqueID [16]byte

		0x01: "System Management URL",
		0x02: "System Name",
		0x03: "System Ping Address",
		0x04: "Component Management URL",
		0x05: "Component Name",
		0x06: "Component Ping Address",
		0x07: "System Unique ID",
	}
	s, ok := m[t]
	if ok {
		return s
	}
	return ""
}

// FRU: 18.4 Management Access Record (Record Type 0x03)
type FRURecordTypeManagementAccess struct {
	SubRecordType ManagementAccessSubRecordType
	Data          []byte // the size is MultiRecord.TypeLength.Length() - 1
}

func (f *FRURecordTypeManagementAccess) Unpack(msg []byte) error {
	if len(msg) < 1 {
		return ErrUnpackedDataTooShort
	}
	f.SubRecordType = ManagementAccessSubRecordType(msg[0])
	f.Data, _, _ = unpackBytes(msg, 1, len(msg)-1)

	return nil
}

// FRU: 18.5 Base Compatibility Record (Record Type 0x04)
type FRURecordTypeBaseCompatibility struct {
	ManufacturerID         uint32
	EntityID               EntityID
	CompatibilityBase      uint8
	CompatibilityCodeStart uint8
	CodeRangeMask          uint8
}

func (f *FRURecordTypeBaseCompatibility) Unpack(msg []byte) error {
	if len(msg) < 7 {
		return ErrUnpackedDataTooShort
	}
	f.ManufacturerID, _, _ = unpackUint24L(msg, 0)
	f.EntityID = EntityID(msg[3])
	f.CompatibilityBase = msg[4]
	f.CompatibilityCodeStart = msg[5]
	f.CodeRangeMask = msg[6]
	return nil
}

// FRU: 18.6 Extended Compatibility Record (Record Type 0x05)
type FRURecordTypeExtendedCompatiblityRecord struct {
	ManufacturerID         uint32
	EntityID               EntityID
	CompatibilityBase      uint8
	CompatibilityCodeStart uint8
	CodeRangeMask          uint8
}

func (f *FRURecordTypeExtendedCompatiblityRecord) Unpack(msg []byte) error {
	if len(msg) < 7 {
		return ErrUnpackedDataTooShort
	}
	f.ManufacturerID, _, _ = unpackUint24L(msg, 0)
	f.EntityID = EntityID(msg[3])
	f.CompatibilityBase = msg[4]
	f.CompatibilityCodeStart = msg[5]
	f.CodeRangeMask = msg[6]
	return nil
}

// FRU: 18.7 OEM Record (Record Types 0xC0-0xFF)
type FRURecordTypeOEM struct {
	ManufacturerID uint32
	Data           []byte
}

func (f *FRURecordTypeOEM) Unpack(msg []byte) error {
	if len(msg) < 3 {
		return ErrUnpackedDataTooShort
	}
	f.ManufacturerID, _, _ = unpackUint24L(msg, 0)
	f.Data, _, _ = unpackBytes(msg, 3, len(msg)-3)
	return nil
}

// getFRUTypeLengthField return a field data bytes whose length is determined by
// a TypeLength byte. The offset index SHOULD points to the TypeLength field.
func getFRUTypeLengthField(fruData []byte, offset uint16) (nextOffset uint16, typeLength TypeLength, fieldData []byte, err error) {
	if len(fruData) < int(offset+1) {
		err = ErrUnpackedDataTooShort
		return
	}

	typeLength = TypeLength(fruData[offset])
	length := typeLength.Length()
	if len(fruData) < int(offset)+int(length)+1 {
		err = ErrUnpackedDataTooShort
		return
	}

	dataStart := int(offset) + 1
	dataEnd := dataStart + int(length)
	fieldData = fruData[dataStart:dataEnd]
	nextOffset = offset + uint16(length) + 1
	return
}

// getFRUCustomUnusedChecksumFields is a helper function to get
// custom, unused, and checksum these three fields from fru data.
// The offset SHOULD points to the start of the custom area fields.
func getFRUCustomUnusedChecksumFields(fruData []byte, offset uint16) (custom [][]byte, unused []byte, checksum uint8, err error) {
	if len(fruData) < int(offset+1) {
		err = ErrUnpackedDataTooShort
		return
	}

	for {
		if fruData[offset] == FRUAreaFieldsEndMark {
			break
		}
		nextOffset, _, fieldData, e := getFRUTypeLengthField(fruData, offset)
		if e != nil {
			err = fmt.Errorf("getFRUTypeLengthField failed, err: %s", e)
			return
		}
		offset = nextOffset

		if len(fieldData) == 0 {
			break
		}
		custom = append(custom, fieldData)
	}

	unusedBytesOffset := int(offset) + 1
	unusedBytesLen := len(fruData) - int(offset) - 2
	unused, _, _ = unpackBytes(fruData, unusedBytesOffset, unusedBytesLen)
	checksum = fruData[len(fruData)-1]
	return
}
