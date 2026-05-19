package identity

type Registry interface {
	Resolve(sig SemanticSignature) (StableID, error)
}
