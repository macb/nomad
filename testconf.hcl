datacenter = "region"
data_dir = "/tmp/nomad"
server {
    enabled = true
    bootstrap_expect = 3
}
advertise {
    rpc = "192.168.0.12:4647"
    serf = "192.168.0.12:4648"
}

