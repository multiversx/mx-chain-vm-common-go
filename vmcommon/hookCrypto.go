package vmcommon

import (
	"math/big"
)

// Bn128Point point on a curve
type Bn128Point struct {
	X *big.Int
	Y *big.Int
}

// Bn128G2Point point on a curve
type Bn128G2Point struct {
	X1 *big.Int
	X2 *big.Int
	Y1 *big.Int
	Y2 *big.Int
}

// CryptoHook interface for VM krypto functions
type CryptoHook interface {
	// Sha256 cryptographic function
	Sha256(str string) (string, error)

	// Keccak256 cryptographic function
	Keccak256(str string) (string, error)

	// Ripemd160 cryptographic function
	Ripemd160(str string) (string, error)

	// EcdsaRecover cryptographic function
	EcdsaRecover(hash string, v *big.Int, r string, s string) (string, error)

	// Sha256 cryptographic function
	Bn128valid(p Bn128Point) (bool, error)

	// Bn128g2valid
	Bn128g2valid(p Bn128G2Point) (bool, error)

	// Bn128add
	Bn128add(p1 Bn128Point, p2 Bn128Point) (Bn128Point, error)

	// Bn128mul
	Bn128mul(k *big.Int, p Bn128Point) (Bn128Point, error)

	// Bn128ate
	Bn128ate(l1 []Bn128Point, l2 []Bn128G2Point) (bool, error)
}
