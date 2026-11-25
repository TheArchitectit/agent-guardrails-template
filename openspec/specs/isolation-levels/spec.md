# isolation-levels Specification

Status: CLOSED (EXERCISED 3)

## Purpose

Enforce named sandbox isolation levels, each with a declared capability matrix over filesystem, network, and process limits, failing closed on a container the host cannot contain.

## Requirements

### Requirement: named levels with capability matrices

The platform SHALL publish exactly the named levels with their filesystem, network, process, secret, and persistence rights. A workload SHALL NOT receive rights outside its level's matrix.

#### Scenario: observer attempts write

- **GIVEN** a workload at observer level
- **WHEN** it attempts a filesystem write
- **THEN** the write SHALL fail and the attempt SHALL be recorded.

#### Scenario: restricted attempts egress

- **GIVEN** a restricted workload
- **WHEN** it attempts a network connection outside its declared egress
- **THEN** the connection SHALL fail and the attempt SHALL be evidence.

### Requirement: declared level enforcement

A workload SHALL declare its required level in the bundle. The runtime SHALL enforce that level or SHALL refuse to run the workload.

#### Scenario: host cannot enforce contained

- **GIVEN** a host lacking the container mechanism
- **WHEN** a contained-level workload is requested
- **THEN** the runtime SHALL refuse with an explicit diagnostic and SHALL NOT run it weaker.

### Requirement: fail-closed containment

Any containment mechanism fault SHALL stop the workload, produce an ERROR verdict, and emit evidence naming the fault.

#### Scenario: runtime enforcement error

- **GIVEN** a seccomp or mount failure mid-workload
- **WHEN** the fault occurs
- **THEN** execution SHALL halt and the run SHALL NOT produce a passing verdict.
