#temp
0 - Button reset, act high
1 - Button WPS, act high
16 - Button Check, act low
14 - OUT, USB enable, active high

3 - LED link green
2 - status LED blue
13 - status LED green
15 - status LED red

10 - CNCNT Link - input, active high

11 - 
    upd:
        die - set IN, unexport
        start: export, set OUT (low)
        end: set IN (high), unexport

    mount:
        export
        set dir high (IN)
        set val 0
    umount:
        set to dir high - IN
        unexport


GPIOs 0-17, ath79:
 gpio-10  (sysfs               ) in  hi
 gpio-11  (sysfs               ) out lo

