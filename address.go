package fins

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsedAddress 表示从字符串解析得到的 PLC 地址。
//
// 支持：
// - 字地址：D100, CIO100, WR200, HR300, A0, T0, C0
// - 位地址：CIO0.00, WR10.15, HR200.01, A5.0
//
// 约束：
// - bit 仅支持 CIO/WR/HR/A
// - bit 位号范围 0~15
// - 地址范围 0~65535
//
// 注意：本类型只负责“字符串 -> areaCode/address/bitNo”的映射，不包含 count/byteCount。
// count/byteCount 由调用方方法入参提供。
//
// 参考常量：[`MemAreaD`](constants.go:53)、[`MemAreaCIO`](constants.go:48)
// 错误：[`ErrInvalidAddress`](types.go:16)
//
//nolint:revive // 该结构体字段名与协议字段保持一致
type ParsedAddress struct {
	AreaCode byte
	Address  uint16
	BitNo    byte
	IsBit    bool

	Original string
}

// ParseAddress 解析 PLC 地址字符串。
func ParseAddress(s string) (*ParsedAddress, error) {
	orig := strings.TrimSpace(s)
	if orig == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidAddress)
	}

	upper := strings.ToUpper(orig)
	upper = strings.ReplaceAll(upper, " ", "")

	areaCode, rest, ok := parseAreaPrefix(upper)
	if !ok {
		return nil, fmt.Errorf("%w: unknown area prefix in %q", ErrInvalidAddress, orig)
	}
	if rest == "" {
		return nil, fmt.Errorf("%w: missing address in %q", ErrInvalidAddress, orig)
	}

	// bit address
	if strings.Contains(rest, ".") {
		if !isBitArea(areaCode) {
			return nil, fmt.Errorf("%w: area %s does not support bit address (%q)", ErrInvalidAddress, GetMemoryAreaName(areaCode), orig)
		}

		parts := strings.Split(rest, ".")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("%w: invalid bit address format %q", ErrInvalidAddress, orig)
		}

		addr, err := parseUint16Dec(parts[0])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid address number in %q: %v", ErrInvalidAddress, orig, err)
		}
		bit, err := parseUint8Dec(parts[1])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid bit number in %q: %v", ErrInvalidAddress, orig, err)
		}
		if bit > 15 {
			return nil, fmt.Errorf("%w: bit out of range (0-15) in %q", ErrInvalidAddress, orig)
		}

		return &ParsedAddress{
			AreaCode: areaCode,
			Address:  addr,
			BitNo:    byte(bit),
			IsBit:    true,
			Original: orig,
		}, nil
	}

	// word address
	addr, err := parseUint16Dec(rest)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address number in %q: %v", ErrInvalidAddress, orig, err)
	}

	return &ParsedAddress{
		AreaCode: areaCode,
		Address:  addr,
		BitNo:    0,
		IsBit:    false,
		Original: orig,
	}, nil
}

func parseAreaPrefix(s string) (areaCode byte, rest string, ok bool) {
	// 注意顺序：优先匹配更长的前缀
	switch {
	case strings.HasPrefix(s, "CIO"):
		return MemAreaCIO, s[len("CIO"):], true
	case strings.HasPrefix(s, "WR"):
		return MemAreaWR, s[len("WR"):], true
	case strings.HasPrefix(s, "HR"):
		return MemAreaHR, s[len("HR"):], true
	case strings.HasPrefix(s, "D"):
		return MemAreaD, s[len("D"):], true
	case strings.HasPrefix(s, "A"):
		return MemAreaA, s[len("A"):], true
	case strings.HasPrefix(s, "T"):
		return MemAreaT, s[len("T"):], true
	case strings.HasPrefix(s, "C"):
		return MemAreaC, s[len("C"):], true
	default:
		return 0, "", false
	}
}

func isBitArea(areaCode byte) bool {
	switch areaCode {
	case MemAreaCIO, MemAreaWR, MemAreaHR, MemAreaA:
		return true
	default:
		return false
	}
}

func parseUint16Dec(s string) (uint16, error) {
	u, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(u), nil
}

func parseUint8Dec(s string) (uint8, error) {
	u, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(u), nil
}
