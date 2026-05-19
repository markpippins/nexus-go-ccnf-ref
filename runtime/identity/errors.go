package identity

import "fmt"

type StableID string

type SemanticSignature string

type OriginSeed string

var ErrNotFound = fmt.Errorf("identity: SemanticSignature not found in registry")
var ErrAlreadyExists = fmt.Errorf("identity: SemanticSignature already registered")
var ErrAmbiguous = fmt.Errorf("identity: multiple OriginSeeds match same SemanticSignature")
