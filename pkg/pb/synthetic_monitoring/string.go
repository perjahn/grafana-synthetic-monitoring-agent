// Code generated by "enumer -type=CheckType -trimprefix=CheckType -transform=lower -output=string.go"; DO NOT EDIT.

package synthetic_monitoring

import (
	"fmt"
	"strings"
)

const _CheckTypeName = "dnshttppingtcptraceroutescriptedmultihttpgrpcbrowser"

var _CheckTypeIndex = [...]uint8{0, 3, 7, 11, 14, 24, 32, 41, 45, 52}

const _CheckTypeLowerName = "dnshttppingtcptraceroutescriptedmultihttpgrpcbrowser"

func (i CheckType) String() string {
	if i < 0 || i >= CheckType(len(_CheckTypeIndex)-1) {
		return fmt.Sprintf("CheckType(%d)", i)
	}
	return _CheckTypeName[_CheckTypeIndex[i]:_CheckTypeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _CheckTypeNoOp() {
	var x [1]struct{}
	_ = x[CheckTypeDns-(0)]
	_ = x[CheckTypeHttp-(1)]
	_ = x[CheckTypePing-(2)]
	_ = x[CheckTypeTcp-(3)]
	_ = x[CheckTypeTraceroute-(4)]
	_ = x[CheckTypeScripted-(5)]
	_ = x[CheckTypeMultiHttp-(6)]
	_ = x[CheckTypeGrpc-(7)]
	_ = x[CheckTypeBrowser-(8)]
}

var _CheckTypeValues = []CheckType{CheckTypeDns, CheckTypeHttp, CheckTypePing, CheckTypeTcp, CheckTypeTraceroute, CheckTypeScripted, CheckTypeMultiHttp, CheckTypeGrpc, CheckTypeBrowser}

var _CheckTypeNameToValueMap = map[string]CheckType{
	_CheckTypeName[0:3]:        CheckTypeDns,
	_CheckTypeLowerName[0:3]:   CheckTypeDns,
	_CheckTypeName[3:7]:        CheckTypeHttp,
	_CheckTypeLowerName[3:7]:   CheckTypeHttp,
	_CheckTypeName[7:11]:       CheckTypePing,
	_CheckTypeLowerName[7:11]:  CheckTypePing,
	_CheckTypeName[11:14]:      CheckTypeTcp,
	_CheckTypeLowerName[11:14]: CheckTypeTcp,
	_CheckTypeName[14:24]:      CheckTypeTraceroute,
	_CheckTypeLowerName[14:24]: CheckTypeTraceroute,
	_CheckTypeName[24:32]:      CheckTypeScripted,
	_CheckTypeLowerName[24:32]: CheckTypeScripted,
	_CheckTypeName[32:41]:      CheckTypeMultiHttp,
	_CheckTypeLowerName[32:41]: CheckTypeMultiHttp,
	_CheckTypeName[41:45]:      CheckTypeGrpc,
	_CheckTypeLowerName[41:45]: CheckTypeGrpc,
	_CheckTypeName[45:52]:      CheckTypeBrowser,
	_CheckTypeLowerName[45:52]: CheckTypeBrowser,
}

var _CheckTypeNames = []string{
	_CheckTypeName[0:3],
	_CheckTypeName[3:7],
	_CheckTypeName[7:11],
	_CheckTypeName[11:14],
	_CheckTypeName[14:24],
	_CheckTypeName[24:32],
	_CheckTypeName[32:41],
	_CheckTypeName[41:45],
	_CheckTypeName[45:52],
}

// CheckTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func CheckTypeString(s string) (CheckType, error) {
	if val, ok := _CheckTypeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _CheckTypeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to CheckType values", s)
}

// CheckTypeValues returns all values of the enum
func CheckTypeValues() []CheckType {
	return _CheckTypeValues
}

// CheckTypeStrings returns a slice of all String values of the enum
func CheckTypeStrings() []string {
	strs := make([]string, len(_CheckTypeNames))
	copy(strs, _CheckTypeNames)
	return strs
}

// IsACheckType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i CheckType) IsACheckType() bool {
	for _, v := range _CheckTypeValues {
		if i == v {
			return true
		}
	}
	return false
}
