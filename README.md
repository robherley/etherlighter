# etherlighter 🔦

Control your UniFi [Etherlighting enabled devices](https://www.ui.com/switching/professional-max) beyond the default modes.

![switch](/docs/img/switch.png)
![web interface](/docs/img/ui.png)

## Disclaimers

> [!CAUTION]
> Please be aware of Ubiquiti's Terms of Service and End User License Agreement before using this software.

```
********************************* NOTICE **********************************
* By logging in to, accessing, or using any Ubiquiti product, you are     *
* signifying that you have read our Terms of Service (ToS) and End User   *
* License Agreement (EULA), understand their terms, and agree to be       *
* fully bound to them. The use of SSH (Secure Shell) can potentially      *
* harm Ubiquiti devices and result in lost access to them and their data. *
* By proceeding, you acknowledge that the use of SSH to modify device(s)  *
* outside of their normal operational scope, or in any manner             *
* inconsistent with the ToS or EULA, will permanently and irrevocably     *
* void any applicable warranty.                                           *
***************************************************************************
```

And for my disclaimer: By running this software, the author of etherlighter is not responsible for any damage caused by the use of this software. This is for educational purposes only.

## Setup

1. Make sure you have a UniFi device with Etherlighting adopted into your network.
1. Find the IP address of the device you want to connect to. This can be found in the "UniFi Devices" tab in Unifi Network.
1. Get your device authentication credentials under Settings > System > Advanced.
1. (Optional) Add an RSA SSH key for passwordless authentication.
1. Build it: `go build`
2. Run it: `./etherlighter -device <ip> -user <username>`
   - For usage: `./etherlighter -help`

## How does it work

This program will establish an SSH connection to your UniFi device and execute commands that adjust the Etherlighting options. To achieve this, most of the operations involve writing to a procfs file (`/proc/led/*`) with different reverse engineered configurations. Searching the device's filesystem, you can find references to different scripts (e.g. in `/etc/rc.d`) that write to files in that directory.

## References

Inspired by:
- https://github.com/adamjezek98/ubnt-etherlighting
