package core

type ChainCfg struct {
	Chain string   `toml:"chain" json:"chain"`
	Rpcs  []string `toml:"rpcs" json:"rpcs"`
	Wss   []string `toml:"wss" json:"wss"`
}

type ChainsCfg struct {
	Chains map[string]ChainCfg `toml:"chains"`
}

type Vault struct {
	Id      string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Chain   string `protobuf:"bytes,2,opt,name=chain,proto3" json:"chain,omitempty"`
	Address string `protobuf:"bytes,3,opt,name=address,proto3" json:"address,omitempty"`
	Token   string `protobuf:"bytes,4,opt,name=token,proto3" json:"token,omitempty"`
}
