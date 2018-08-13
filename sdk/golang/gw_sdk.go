package golang

type Node struct{
	Name []byte
	Host []byte
	Port uint8
	Status uint8
	HC *HealthCheck
}
type Service struct{
	Name []byte
	Node []*Node
}
type Router struct{
	Name []byte
	Frontend []byte
	Backend []byte
	Service *Service
}
type HealthCheck struct{

}

