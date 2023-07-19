# juju-dqlite-backstop

This is a tool to repair broken dqlite leader in a Juju instance. It is only of
interest to people responsible for Juju installations that are experiencing
catastrophic Juju instance outages. It should not be used casually. Improper
use could lead to irreverible damage to Juju deployments.

juju-dqlite-backstop is typically run on the last controller machine that was
part of a HA cluster. The tool will attempt to repair the dqlite leader and
restore the cluster to a healthy state. This can happen if the leader has
become a standby node and not considered a voter in the cluster election.

## Usage

To build the tool, run `make clean build`. This will produce a binary called
`juju-dqlite-backstop` in the bin directory. It will create a standalone
statically compiled binary that will compile in the version of dqlite that is
present in the `scripts/dqlite/scripts/env.sh` file.

Running the tool requires transferring the binary to the controller machine
either via `scp` or `lxc file push`. Once the binary is on the controller
machine, run the following command:

```
./juju-dqlite-backstop machine-${machine-number}
```

Where `${machine-number}` is the machine number of the controller machine that
is experiencing the dqlite leader issue. The tool will attempt to repair the
dqlite leader and restore the cluster to a healthy state.

Following the running of the tool, you will be required to run on the controller
machine to restart the agent:

```
systemctl restart juju-machine-${machine-numer}.service
```
