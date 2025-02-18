---
name: Bug report
about: Create a report to help us improve
title: ''
labels: bug
assignees: ''
body:
  - type: markdown
    attributes:
      value: |
        Thanks for taking the time to fill out this bug report!
  - type: textarea
    id: expected
    attributes:
      label: Expected behavior
      description: What is expected behavior? How SDK should behave?
      placeholder: Tell us what you should get!
    validations:
      required: true
  - type: textarea
    id: expected
    attributes:
      label: Actual behavior
      description: What actually happened?
      placeholder: Tell us what you see!
    validations:
      required: true
  - type: dropdown
    id: version
    attributes:
      label: Version
      description: What version of our software are you running?
      options:
        - 0.1.0
      default: 0
    validations:
      required: true
  - type: textarea
    id: logs
    attributes:
      label: Relevant log output
      description: Please copy and paste any relevant log output. This will be automatically formatted into code, so no need for backticks.
      render: shell
---
