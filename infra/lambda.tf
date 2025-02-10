variable "lambda_function_name" {
  default = "cheerleader_proxy_api"
}

resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${var.lambda_function_name}"
  retention_in_days = 7
  skip_destroy      = false
}

resource "aws_lambda_function" "lambda_func" {
  filename         = data.archive_file.lambda_zip.output_path
  function_name    = var.lambda_function_name
  handler          = "app"
  source_code_hash = data.archive_file.lambda_zip.output_sha256
  runtime          = "provided.al2023"
  role             = aws_iam_role.lambda_exec.arn

  environment {
    variables = {
      DDB_TABLE = aws_dynamodb_table.score_table.name
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.lambda_logs
  ]
}

resource "aws_iam_policy" "lambda_policy" {
  name = "LambdaPolicy"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:BatchGetItem",
          "dynamodb:PutItem",
          "dynamodb:Query",
        ]
        Resource = [
          aws_dynamodb_table.score_table.arn,
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:Query",
        ]
        Resource = [
          "${aws_dynamodb_table.score_table.arn}/index/*",
        ]
      },
      {
        Effect = "Allow",
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ],
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role" "lambda_exec" {
  name_prefix = local.app_id

  assume_role_policy = jsonencode({
    Version: "2012-10-17",
    Statement: [
      {
        Action: "sts:AssumeRole",
        Principal: {
          Service: "lambda.amazonaws.com"
        },
        Effect: "Allow",
        Sid: ""
      }
    ]
  })
}

resource "aws_iam_policy_attachment" "role_attach" {
  name       = "policy-${local.app_id}"
  roles      = [aws_iam_role.lambda_exec.id]
  policy_arn = aws_iam_policy.lambda_policy.arn
}
