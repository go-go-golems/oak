resource "aws_ecs_cluster" "go_webserver_test_cluster" {
  name = "go_webserver_test_cluster"
}

resource "aws_ecs_task_definition" "go_webserver_test_task_definition" {
  family = "go_webserver_test_task_family"
  container_definitions = jsonencode([
    {
      "name" : "go_webserver_test_container",
      "image" : "${aws_ecr_repository.go_webserver_test_repository.repository_url}:latest",
      "portMappings" : [
        {
          "containerPort" : 80,
          "hostPort" : 80,
          "protocol" : "tcp"
        }
      ],
      "logConfiguration" : {
        "logDriver" : "awslogs",
        "options" : {
          "awslogs-group" : "${aws_cloudwatch_log_group.go_webserver_test_log_group.name}",
          "awslogs-region" : "${data.aws_region.current.name}",
          "awslogs-stream-prefix" : "go_webserver_test_container"
        }
      },
      healthCheck = {
        command     = ["CMD-SHELL", "curl -f http://localhost:80/health || exit 1"]
        interval    = 30
        timeout     = 5
        startPeriod = 60
        retries     = 3
      }
    }
  ])
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"
  network_mode             = "awsvpc"
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
}

resource "aws_ecs_service" "go_webserver_test_service" {
  name            = "go_webserver_test_service"
  cluster         = aws_ecs_cluster.go_webserver_test_cluster.arn
  task_definition = aws_ecs_task_definition.go_webserver_test_task_definition.arn
  desired_count   = 2
  launch_type     = "FARGATE"

#   depends_on = [
#     aws_alb_listener.go_webserver_test_listener,
#   ]

  deployment_minimum_healthy_percent = 80
  deployment_maximum_percent         = 150

  network_configuration {
    security_groups = [
    aws_security_group.ecs_security_group.id] # Setting the security group

    subnets = [
    aws_subnet.private-subnet-1.id, 
    aws_subnet.private-subnet-2.id,
    ]

  }
  load_balancer {
    target_group_arn = aws_alb_target_group.go_webserver_test_target_group.arn
    container_name   = "go_webserver_test_container"
    container_port   = 80
  }
}

resource "aws_iam_role" "ecs_execution_role" {
  name = "ecs_execution_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
        Sid = ""
      }
    ]
  })
}

resource "aws_security_group" "ecs_security_group" {
  name        = "ecs_security_group"
  description = "Allow all inbound traffic"
  vpc_id      = aws_vpc.go_webserver_test_vpc.id

  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    security_groups = [
      aws_security_group.go_webserver_test_sg.id
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_role_policy_attachment" "ecs_execution_role_policy_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.ecs_execution_role.name
}
