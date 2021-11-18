package vmcommon

const lengthOfEsdtLocalRoles = 64

type Bits byte

const (
	RoleMint = 1 << iota
	RoleBurn
	RoleNFTCreate
	RoleNFTAddQuantity
	RoleNFTBurn
)

func Set(b, flag int) int  { return b | flag }
func Has(b, flag int) bool { return b&flag != 0 }

// EsdtLocalRoles represents smart contract code metadata
type EsdtLocalRoles struct {
	Mint           bool
	Burn           bool
	NFTCreate      bool
	NFTAddQuantity bool
	NFTBurn        bool
}

// EsdtLocalRolesFromBytes creates a roles object from bytes
func EsdtLocalRolesFromBytes(bytes []byte) EsdtLocalRoles {
	if len(bytes) != lengthOfEsdtLocalRoles {
		return EsdtLocalRoles{}
	}

	return EsdtLocalRoles{
		Mint:           Has(fromByteArray(bytes), RoleMint),
		Burn:           Has(fromByteArray(bytes), RoleBurn),
		NFTCreate:      Has(fromByteArray(bytes), RoleNFTCreate),
		NFTAddQuantity: Has(fromByteArray(bytes), RoleNFTAddQuantity),
		NFTBurn:        Has(fromByteArray(bytes), RoleNFTBurn),
	}
}

// ToBytes converts the roles to bytes
func (roles *EsdtLocalRoles) ToBytes() []byte {
	value := 0

	if roles.Mint {
		value = Set(value, RoleMint)
	}
	if roles.Burn {
		value = Set(value, RoleBurn)
	}
	if roles.NFTCreate {
		value = Set(value, RoleNFTCreate)
	}
	if roles.NFTAddQuantity {
		value = Set(value, RoleNFTAddQuantity)
	}
	if roles.NFTBurn {
		value = Set(value, RoleNFTBurn)
	}

	return toByteArray(value)
}

func fromByteArray(bytes []byte) int {
	newInt := ((int(bytes[0]) & 0xFF) << 56) |
		((int(bytes[1]) & 0xFF) << 48) |
		((int(bytes[2]) & 0xFF) << 40) |
		((int(bytes[3]) & 0xFF) << 32) |
		((int(bytes[4]) & 0xFF) << 24) |
		((int(bytes[5]) & 0xFF) << 16) |
		((int(bytes[6]) & 0xFF) << 8) |
		((int(bytes[7]) & 0xFF) << 0)
	return newInt
}

func toByteArray(value int) []byte {
	newByteArray := []byte{
		byte(value >> 56),
		byte(value >> 48),
		byte(value >> 40),
		byte(value >> 32),
		byte(value >> 24),
		byte(value >> 16),
		byte(value >> 8),
		byte(value),
	}
	return newByteArray
}
