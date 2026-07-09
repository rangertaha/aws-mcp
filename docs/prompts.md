# Prompts (workflows)

MCP has no dedicated "workflow" primitive, so multi-step flows are shipped as **prompts**: user-invoked, parameterized templates that guide the model through a sequence of tool calls. MCP clients surface these as slash commands automatically (e.g. in Claude Code and Claude Desktop).

| Prompt | Arguments | What it does |
| ------ | --------- | ------------ |
| `survey_bucket` | `bucket` | List an S3 bucket's objects via `aws_invoke` (`s3.ListObjectsV2`) and summarize object count/size |

`survey_bucket` doesn't call S3 itself — it renders a short instruction telling the model which `aws_invoke` call to make and what to report back, the same way any other prompt-driven workflow would use the [meta tools](index.md).

## Next: how the tools actually work

See [Architecture](architecture.md) for how `aws_invoke` and friends are implemented.
