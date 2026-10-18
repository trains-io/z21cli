## The z21 Command Line Interface

A command line utility to interact with and manage z21 command stations.

### Features

- **Context** — named endpoints, sessions, and configuration file management
- **Version** — device identity, firmware, hardware, and protocol version
- **Track** — track power, emergency stop, and driving locomotives
- **System** — connectivity, status, LAN discovery, and CAN detectors
- **Monitor** — broadcast flag control, named subscriptions, and live event streaming

### Installation

Releases are [published to GitHub](https://github.com/trains-io/z21cli/releases) where Zip, RPMs and DEBs for various operating systems can be found.

#### Installation via go install

The `z21` CLI can be installed directly via `go install`. To install the latest version:

```bash
go install github.com/trains-io/z21cli@latest
```

To install a specific release:

```sh
go install github.com/trains-io/z21cli@v0.1.0
```

### Commands

Run `z21 --help` for the full list. Commands are grouped as follows:

| Group | Commands |
|-------|----------|
| **Version** | `code`, `firmware`, `hwinfo`, `version` |
| **Context** | `context` (`ctx`), `logoff` |
| **Track** | `loco`, `power`, `stop` |
| **System** | `can`, `discover`, `ping`, `serial`, `status`, `system` |
| **Monitor** | `broadcast` (`bc`), `monitor` (`mon`), `subscribe` (`sub`) |

`info` shows aggregated device information in a single request.

Most commands use the current context. Pass `--host` (and optionally `--port`) to query a command station directly without a saved session.

### Context commands

The `z21` CLI supports multiple named configurations. Contexts are stored in `~/.z21_contexts.json`.

Add a context for a local simulator:

```bash
z21 context add sim --host 127.0.0.1
```

Output:

```bash
z21 Configuration Context "sim"

Host:        127.0.0.1:21105
Config file: /home/user/.z21_contexts.json
```

Add another context and select it as the default:

```bash
z21 context add demo --host 192.168.2.6 --use --description "z21 device reserved for demo"
```

Output:

```bash
z21 Configuration Context "demo"

Description: z21 device reserved for demo
Host:        192.168.2.6:21105
Config file: /home/user/.z21_contexts.json
```

List saved contexts (`(*)` marks the current context):

```bash
z21 context ls
```

Output:

```bash
Known contexts:

    NAME ENDPOINT          DESCRIPTION
-------------------------------------------------------
( ) sim  127.0.0.1:21105
(*) demo 192.168.2.6:21105 z21 device reserved for demo

Config file: /home/user/.z21_contexts.json
```

Switch context:

```bash
z21 ctx use sim
```

Show the current context:

```bash
z21 ctx show
```

Output:

```bash
z21 Configuration Context "demo"

Host:        192.168.2.6:21105
Description: z21 device reserved for demo
Session:     172.20.236.167:38991
Config file: /home/user/.z21_contexts.json
```

When no session exists yet, `Session` is shown as `-`.

#### Session management

The z21 command station tracks each client by local IP address and UDP source port. Use `z21 logoff` to send `LAN_LOGOFF` and clear the saved session for the current context.

Clear session data for the current context (sends logoff when a session is active):

```sh
z21 ctx reset
```

Use `z21 context --help` for add, remove, list, show, use, and reset.

### Track commands

Control track power and emergency stop on the current context:

```sh
z21 power on
z21 power off
z21 power status
z21 stop
```

#### Locomotives

Read and drive a locomotive by address (1–9999). Each command shows the loco's state afterwards:

```sh
z21 loco info 3
z21 loco drive 3 40                         # speed step 40, direction and speed steps kept
z21 loco drive 3 10 --reverse --steps 28    # 14, 28 or 128 speed steps
z21 loco func 3 0=on 1=toggle 2=off         # functions F0–F31
z21 loco stop 3                             # emergency stop for this loco only
```

Output:

```sh
LOCO        STATE
-------------------
Address     3
Format      DCC
Speed steps 128
Direction   forward
Speed       40/126
Functions   F0
Busy        no
```

Speed is a speed step: 0 stops, at most 14, 28 or 126. Asking for a loco subscribes the session to its changes (at most 16 locos); with `XPRESSNET_UPDATES` subscribed, `z21 monitor` shows them as `[LOCO]` lines. `ALL_LOCO_INFO_UPDATES` shows the changes of every loco.

### System commands

#### Connectivity and status

Test connectivity (queries serial number and hardware info):

```bash
z21 ping
```

Output:

```bash
pong from 127.0.0.1:21105 (serial=999620936, hw=0x00000201 fw=0x00000143)
```

Track status flags:

```bash
z21 status
```

Output:

```bash
 TRACK             STATUS (0X00)
---------------------------------
 Emergency Stop    OFF
 Track Voltage     ON
 Short Circuit     OFF
 Programming Mode  INACTIVE
```

System measurements (current, temperature, voltage):

```bash
z21 system
```

Output:

```bash
METRIC           VALUE
----------------------------------------------------------------------------------------------
Main current     0 mA
Prog current     0 mA
Filtered main    0 mA
Temperature      0 °C
Supply voltage   0 mV
VCC voltage      0 mV
Central state    0x00 (ok)
Central state ex 0x00 (ok)
Capabilities     0x7f (DCC, MM, RailCom, loco commands, accessory commands, detector commands)
```

#### Device identity

Individual fields are also available:

```bash
z21 serial
z21 version
z21 firmware
z21 hwinfo
z21 code
```

Or fetch everything in one combined request:

```bash
z21 info
```

#### LAN discovery

Scan the local network for z21 command stations (does not require a context):

```bash
z21 discover
```

#### CAN occupancy detectors

Subscribe to detector events first (see Monitor commands), then discover or inspect devices:

```bash
z21 sub add CAN_DETECTOR_UPDATES
z21 can discover
z21 can info 0xdb04
```

Example discovery output:

```bash
Discover CAN devices (timeout: 2s) ...
 NETID   ADDR  PORT(S)
-----------------------
 0xDB04  31    1-8
```

Example device output:

```bash
Device: 0xDB04 (address: 31)
 PORT  STATUS
--------------
 1     free
 2     free
 3     free
 4     free
 5     free
 6     free
 7     free
 8     free
```

### Monitor commands

Broadcast flags control which UDP events the z21 sends to this client (per IP address and local UDP port).

#### Named subscriptions

Subscribe and unsubscribe by name on the current session:

```bash
z21 sub add SYSTEM_UPDATES
z21 sub ls
z21 sub rm SYSTEM_UPDATES
```

Example `sub add` output:

```bash
Subscribed to "SYSTEM_UPDATES"
```

Example `sub ls` output:

```bash
Broadcast mask: 0x00000101

Active subscriptions:

XPRESSNET_UPDATES: Track power, programming mode, short circuit, stop, locomotive and turnout messages.
SYSTEM_UPDATES:    Z21 system status such as track voltage.
```

Run `z21 sub --help` for all supported subscription names.

#### Raw broadcast flags

Set or inspect the full 32-bit mask directly:

```bash
z21 bc get
z21 bc set 0x101
```

Use `z21 bc set` to replace the entire mask; use `z21 sub add` to enable one named group without clearing others.

#### Live monitoring

After subscribing, stream events on the current session:

```bash
z21 monitor
```

Output:

```bash
Waiting for z21 events ...
[SYS] Main: 30mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.0V)
[SYS] Main: 36mA  Prog: 0mA   Temp: 27°C  Volt: 20.0V (18.0V)
[LOCO] 3: forward, speed 40/126, functions F0 (LAN_X_LOCO_INFO)
```

Press `Ctrl+C` to stop.

### Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for how to build, run tests, and run against the z21 simulator.

### License

[MIT](./LICENSE)
