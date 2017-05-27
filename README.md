# Protector

Protect (and free) your GitHub branches.  
Greatly inspired by [audit](https://github.com/jessfraz/audit) and [pepper](https://github.com/jessfraz/pepper).

## Usage

```
$> protector -h                                                                                                                                                           

protector - v0.1.0-SNAPSHOT
  -branches value
    	branches to include (as regexp)
  -dry-run
    	do not make any changes, just print out what would have been done
  -free
    	remove branch protection
  -orgs value
    	organizations name to protect
  -repos value
    	repositories fullname to protect (ex: jcgay/maven-color)
  -token string
    	GitHub API token
  -v	print version and exit (shorthand)
  -version
    	print version and exit
```

## Build

### Status

[![Build Status](https://travis-ci.org/jcgay/protector.svg?branch=master)](https://travis-ci.org/jcgay/protector)
[![Code Report](https://goreportcard.com/badge/github.com/jcgay/protector)](https://goreportcard.com/report/github.com/jcgay/protector)
[![Coverage Status](https://coveralls.io/repos/github/jcgay/protector/badge.svg?branch=master)](https://coveralls.io/github/jcgay/protector?branch=master)