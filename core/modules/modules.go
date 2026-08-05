package modules

// Define o contrato estrito (Strategy Pattern) que todos os instaladores devem seguir.
type GRCModule interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetIconSVG() string
	RunSilent() error
}
