#!/usr/bin/env python3

import os
import re
import sys

target = "./cfg.toml"
ports = []

table = None

def handleConfigLine(line):
    if line.startswith('#'):
        return
    global table
    matched = re.match(r'^\[.*\]$', line, re.M | re.I)
    if matched:
        table = line[1:len(line) - 2]
        return
    if table:
        line = line.replace(' ','')
        if table == "proxy" and line.startswith('listen='):
            parsePort(line[7:])
        elif table == "register" and line.startswith('listen='):
            parsePort(line[7:])
        elif table == "cluster.local" and line.startswith('port='):
            parsePort(line[5:])


def parsePort(port):
    p = port[2:len(port) - 2]
    ports.append(p)


def readFile(path, fn):
    try:
        with open(path, "r") as file:
            for line in file.readlines():
                fn(line)
        file.close()
    except FileNotFoundError as err:
        print(f'file not found,err:{err}')
    except PermissionError as err:
        print(f'no permission,err:{err}')


def main():
    if not os.access(target, os.F_OK):
        print('not exists config file %s' % target)
        return
    # read line from file and handle
    readFile(target, handleConfigLine)
    retPorts=""
    for p in ports:
        retPorts +="-p {}:{} ".format(p,p)
    print(retPorts)


if __name__ == "__main__":
    sys.exit(main())
