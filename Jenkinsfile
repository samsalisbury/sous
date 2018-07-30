#!/usr/bin/env groovy
pipeline {
  agent { label 'mesos-qa-uswest2' }
  options {
    // Version 5
    // Set to 1 day to allow people to input whether they want to go to Prod on the Master branch build/deploys
    timeout(time: 1, unit: 'DAYS')
  }
  stages {
    //Immediately send github PR all checks that this pipeline will be checking
    stage('Git statuses') {
      steps {
        githubNotify context: 'Jenkins/Test/Unit', description: 'Unit Tests', status: 'PENDING'
          githubNotify context: 'Jenkins/Test/Static-Check', description: 'Static Check Tests', status: 'PENDING'
          githubNotify context: 'Jenkins/Test/TestSmoke/git/split', description: 'Smoke Git Split Tests', status: 'PENDING'
          githubNotify context: 'Jenkins/Test/TestSmoke/git/simple', description: 'Smoke Git Simple Tests', status: 'PENDING'
          githubNotify context: 'Jenkins/Test/TestOTPL/git/simple', description: 'OTPL Git Simple Tests', status: 'PENDING'
          githubNotify context: 'Jenkins Overall Success', description: 'Pipeline Status', status: 'PENDING'
      }
    }
    stage('Inited Values') {
      steps {
        echo "BUILD_NUMBER=$BUILD_NUMBER"
          echo "BRANCH_NAME=$BRANCH_NAME"
          echo "NODE_NAME=$NODE_NAME"
          echo "NODE_LABELS=$NODE_LABELS"
          echo "BUILD_URL=$BUILD_URL"
          script {
            def notifier = new org.gradiant.jenkins.slack.SlackNotifier()

              env.SLACK_CHANNEL = '#team-eng-sous-bots, #tech-deploy'

              notifier.notifyStart()
          }
      }
    }
    stage('Test'){
      parallel {
        stage('Static') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "static test step"
              sh '''#!/usr/bin/env bash
              set -x
              set -e

              echo "Setting up git identity for test"
              git config --global user.email "sous-internal@opentable.onmicrosoft.com"
              git config --global user.name "Jenkins Run"


              echo $PATH
              PATH=$PATH:/usr/local/go/bin export PATH
              echo $PATH


              echo "Setting up GOPATH"

              mkdir -p godir/src/github.com/opentable
              ln -s $PWD ./godir/src/github.com/opentable/sous
              export GOPATH=$GOPATH:$PWD/godir
              cd $PWD/godir/src/github.com/opentable/sous
              echo $GOPATH


              echo $PATH
              PATH=$PATH:$WORKSPACE/godir/bin export PATH
              echo $PATH

              echo "Running Tests"
              VERBOSE=1 make test-staticcheck

              '''
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Static-Check', description: 'Static Check Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Static-Check', description: 'Static Check Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Unit') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "unit test step"
              sh '''#!/usr/bin/env bash
              set -x
              set -e

              echo $PATH
              PATH=$PATH:/usr/local/go/bin export PATH
              echo $PATH

              echo "Setting up GOPATH"

              mkdir -p godir/src/github.com/opentable
              ln -s $PWD ./godir/src/github.com/opentable/sous
              export GOPATH=$GOPATH:$PWD/godir
              cd $PWD/godir/src/github.com/opentable/sous
              echo $GOPATH

              echo "Running Tests"
              make test-unit

              echo "Generate out file"
              go test -covermode=count -coverprofile=count.out `make export-SOUS_PACKAGES_WITH_TESTS`
              go tool cover -func=count.out
              go tool cover -html=count.out -o coverage.html
              mkdir coverage
              cp coverage.html ./coverage

              '''
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Unit', description: 'Unit Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Unit', description: 'Unit Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Smoke_Simple_Git') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "smoke test TestSomke/git/simple step"
              retry(2) {
                sh '''#!/usr/bin/env bash
                  set -x
                  set -e

                  echo $PATH
                  PATH=$PATH:/usr/local/go/bin export PATH
                  echo $PATH

                  echo "Setting up GOPATH"

                  mkdir -p godir/src/github.com/opentable
                  ln -sfn $PWD $PWD/godir/src/github.com/opentable/sous
                  export GOPATH=$PWD/godir
                  cd $PWD/godir/src/github.com/opentable/sous

                  echo $GOPATH
                  echo $PWD


                  echo "Running Tests"

                  echo "Setting up git identity for test"
                  git config --global user.email "sous-internal@opentable.onmicrosoft.com"
                  git config --global user.name "Jenkins Run"


                  GO_TEST_RUN=TestSmoke/git/simple make test-smoke

                  '''
              }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/TestSmoke/git/simple', description: 'Smoke Git Simple Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/TestSmoke/git/simple', description: 'Smoke Git Simple Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Smoke_Split_Git') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "smoke test TestSomke/git/split step"
              retry(2) {
                sh '''#!/usr/bin/env bash
                  set -x
                  set -e

                  echo $PATH
                  PATH=$PATH:/usr/local/go/bin export PATH
                  echo $PATH

                  echo "Setting up GOPATH"

                  mkdir -p godir/src/github.com/opentable
                  ln -sfn $PWD $PWD/godir/src/github.com/opentable/sous
                  export GOPATH=$PWD/godir
                  cd $PWD/godir/src/github.com/opentable/sous

                  echo $GOPATH
                  echo $PWD


                  echo "Running Tests"

                  echo "Setting up git identity for test"
                  git config --global user.email "sous-internal@opentable.onmicrosoft.com"
                  git config --global user.name "Jenkins Run"


                  GO_TEST_RUN=TestSmoke/git/split make test-smoke

                  '''
              }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/TestSmoke/git/split', description: 'Smoke Git Split Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/TestSmoke/git/split', description: 'Smoke Git Split Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Smoke_TestOTPL_Simple_Git') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "smoke test TestOTPL/git/simple step"
              retry(2) {
                sh '''#!/usr/bin/env bash
                  set -x
                  set -e

                  echo $PATH
                  PATH=$PATH:/usr/local/go/bin export PATH
                  echo $PATH

                  echo "Setting up GOPATH"

                  mkdir -p godir/src/github.com/opentable
                  ln -sfn $PWD $PWD/godir/src/github.com/opentable/sous
                  export GOPATH=$PWD/godir
                  cd $PWD/godir/src/github.com/opentable/sous

                  echo $GOPATH
                  echo $PWD


                  echo "Running Tests"

                  echo "Setting up git identity for test"
                  git config --global user.email "sous-internal@opentable.onmicrosoft.com"
                  git config --global user.name "Jenkins Run"


                  GO_TEST_RUN=TestOTPL/git/simple make test-smoke

                  '''
              }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/TestOTPL/git/simple', description: 'OTPL Git Simple Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/TestOTPL/git/simple', description: 'OTPL Git Simple Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Integration') {
          agent { label 'mesos-qa-uswest2' }
          steps {
            echo "integration test"
              retry(2) {
                sh '''#!/usr/bin/env bash
                  set -x
                  set -e

                  echo $PATH
                  PATH=$PATH:/usr/local/go/bin export PATH
                  echo $PATH

                  echo "Setting up GOPATH"

                  mkdir -p godir/src/github.com/opentable
                  ln -s $PWD ./godir/src/github.com/opentable/sous
                  export GOPATH=$GOPATH:$PWD/godir
                  cd $PWD/godir/src/github.com/opentable/sous
                  echo $GOPATH

                  echo "Running Tests"
                  make test-integration
                  '''
              }
          }
        }
      }
    }
  }
  post {
    always {
      echo 'This will always run'
    }
    success {
      echo 'This will run only if successful'
        script {
          def notifier = new org.gradiant.jenkins.slack.SlackNotifier()

            env.SLACK_CHANNEL = '#team-eng-sous-bots, #tech-deploy'
            env.CHANGE_LIST = 'true'
            env.NOTIFY_SUCCESS = 'true'

            notifier.notifyResult()
        }
      githubNotify context: 'Jenkins Overall Success', description: 'PIPELINE all Passed!!!', status: 'SUCCESS'
    }
    failure {
      echo 'This will run only if failed'
        script {
          def notifier = new org.gradiant.jenkins.slack.SlackNotifier()

            env.SLACK_CHANNEL = '#team-eng-sous-bots, #tech-deploy'
            env.CHANGE_LIST = 'true'
            env.NOTIFY_SUCCESS = 'false'

            notifier.notifyResult()
        }
      githubNotify context: 'Jenkins Overall Success', description: 'PIPELINE FAILED', status: 'FAILURE'
    }
    unstable {
      echo 'This will run only if the run was marked as unstable'
    }
    changed {
      echo 'This will run only if the state of the Pipeline has changed'
        echo 'For example, if the Pipeline was previously failing but is now successful'
    }
  }
}
