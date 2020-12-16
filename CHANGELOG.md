# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

## [19.01.09.00] - 2019-01-09
### Added
- shell.nix for dev env reproducibility
- vendoring protoc-gen-go for code-gen reproducibility
### Fixed
- tls certificate generation (using cfssl)
### Removed
- grpccli stuff

## [18.06.11.00] - 2018-06-11
### Changed
- Server's reneg-sec option is set to 24h

## [18.03.06.00] - 2018-03-06
### Fixed
- All of the connecting user's projects are now fetched.
### Changed
- Deps are now vendored.
- Updated base image to alpine v3.7.
- Now using easy-rsa from alpine's package.
