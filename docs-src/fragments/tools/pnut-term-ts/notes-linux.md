### Serial port access

Add yourself to the `dialout` group so you can open the PropPlug, then log out
and back in for it to take effect:

```sh
sudo usermod -a -G dialout $USER
```

PropPlugs appear as `/dev/ttyUSB*`. List the ones the tool can see with
`{{launcher}} -n`.
