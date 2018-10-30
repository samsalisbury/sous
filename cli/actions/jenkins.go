package actions

import (
	"fmt"
	"os"
	"time"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/valyala/fasttemplate"
)

// Jenkins is used to issue the command to make a new Deployment current for it's SourceID.
type Jenkins struct {
	HTTPClient       restful.HTTPClient
	TargetManifestID sous.ManifestID
	LogSink          logging.LogSink
	User             sous.User
	Cluster          string
	*config.Config
}

// mergeDefaults will take the metadata map, and compare with defaults and merge the two
func (sj *Jenkins) mergeDefaults(metadata map[string]string) map[string]interface{} {
	merge := make(map[string]interface{})

	for k, v := range sj.returnJenkinsDefaultMap() {
		if metaValue, OK := metadata[k]; OK {
			merge[k] = metaValue
		} else {
			merge[k] = v
		}
	}

	return merge
}

// generateJenkinsPipelineString returns string based off the config
func (sj *Jenkins) generateJenkinsPipelineString(jenkinsConfig map[string]interface{}) string {

	templ := sj.returnTemplate()

	t := fasttemplate.New(templ, "{{", "}}")
	pipeLine := t.ExecuteString(jenkinsConfig)

	return pipeLine
}

// saveFile writes out Jenkinsfile to current folder
func (sj *Jenkins) saveFile(pipeline string) error {

	//	dir, err := os.Getwd()
	//	if err != nil {
	//		return err
	//	}

	file, err := os.Create("Jenkinsfile")
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, pipeline)

	return nil
}

// updateMetaData merge back all config to actual metadata
func (sj *Jenkins) updateMetaData(metadata map[string]string, config map[string]interface{}) map[string]string {

	for k, v := range config {
		metadata[k] = v.(string)
	}
	return metadata
}

// Do implements Action on Jenkins.
func (sj *Jenkins) Do() error {

	//Grab metadata from current manifest
	//Merge with Defaults
	//Write out Jenkins
	//Push back metadata

	mani := sous.Manifest{}
	up, err := sj.HTTPClient.Retrieve("/manifest", sj.TargetManifestID.QueryMap(), &mani, nil)
	if err != nil {
		return err
	}

	clusterWithJenkinsConfig := sj.Cluster

	if len(clusterWithJenkinsConfig) < 1 {
		messages.ReportLogFieldsMessageToConsole("Please specify the JenkinsConfigCluster variable in sous config", logging.WarningLevel, sj.LogSink)
		return fmt.Errorf("no config cluster specified")
	}

	currentConfigMap := make(map[string]string)
	if err != nil || mani.Deployments[clusterWithJenkinsConfig].Metadata == nil {
		messages.ReportLogFieldsMessageWithIDs(fmt.Sprintf("Couldn't determine metadata for %s", clusterWithJenkinsConfig), logging.WarningLevel, sj.LogSink, err)
	} else {
		currentConfigMap = mani.Deployments[clusterWithJenkinsConfig].Metadata
	}

	jenkinsConfig := sj.mergeDefaults(currentConfigMap)

	messages.ReportLogFieldsMessageWithIDs("Merged Config Data", logging.ExtraDebug1Level, sj.LogSink, jenkinsConfig)

	jenkinsConfig["SOUS_MANIFEST_ID"] = sj.TargetManifestID.String()
	jenkinsConfig["SOUS_JENKINS_GENERATED_DATE"] = time.Now().Format(time.RFC822)

	jenkinsPipelineString := sj.generateJenkinsPipelineString(jenkinsConfig)

	messages.ReportLogFieldsMessageWithIDs("PipeLine", logging.ExtraDebug1Level, sj.LogSink, jenkinsPipelineString)

	depspec := mani.Deployments[clusterWithJenkinsConfig]
	if depspec.Metadata == nil {
		depspec.Metadata = map[string]string{}
	}

	depspec.Metadata = sj.updateMetaData(depspec.Metadata, jenkinsConfig)
	mani.Deployments[clusterWithJenkinsConfig] = depspec

	if _, err := up.Update(&mani, sj.User.HTTPHeaders()); err != nil {
		return err
	}

	return sj.saveFile(jenkinsPipelineString)
}

func (sj *Jenkins) returnJenkinsDefaultMap() map[string]string {
	return map[string]string{
		"SOUS_JENKINS_GENERATED_DATE":   "",
		"SOUS_MANIFEST_ID":              "",
		"SOUS_DEPLOY_CI":                "YES",
		"SOUS_DEPLOY_PP":                "YES",
		"SOUS_DEPLOY_PROD":              "YES",
		"SOUS_POST_CI_TEST":             "YES",
		"SOUS_POST_CI_TEST_COMMAND":     "make post-ci-test",
		"SOUS_POST_PP_TEST":             "YES",
		"SOUS_POST_PP_TEST_COMMAND":     "make post-pp-test",
		"SOUS_POST_PROD_TEST":           "YES",
		"SOUS_POST_PROD_TEST_COMMAND":   "make post-prod-test",
		"SOUS_INTEGRATION_TEST":         "YES",
		"SOUS_INTEGRATION_TEST_COMMAND": "make integration",
		"SOUS_SMOKE_TEST":               "YES",
		"SOUS_SMOKE_TEST_COMMAND":       "make smoke",
		"SOUS_STATIC_TEST":              "YES",
		"SOUS_STATIC_TEST_COMMAND":      "make static",
		"SOUS_UNIT_TEST":                "YES",
		"SOUS_UNIT_TEST_COMMAND":        "make unit",
		"SOUS_USE_RC":                   "YES",
		"SOUS_VERSIONING_SCHEME":        "semver_timestamp",
		"SOUS_JENKINSPIPELINE_VERSION":  "0.0.1",
		"SOUS_RELEASE_BRANCH":           "master",
		"SOUS_DEPLOY_PROD_QUERY_USER":   "YES",
	}
}

func (sj *Jenkins) returnTemplate() string {

	var template = `#!/usr/bin/env groovy
pipeline {
  agent { label 'mesos-qa-uswest2' }
  // Version {{SOUS_JENKINSPIPELINE_VERSION}}
  // Generated: {{SOUS_JENKINS_GENERATED_DATE}}
  // ManifestID: {{SOUS_MANIFEST_ID}}
  options {
    // Set to 1 day to allow people to input whether they want to go to Prod on the Master branch build/deploys
    timeout(time: 1, unit: 'DAYS')
  }
  parameters {
    string(defaultValue: '{{SOUS_VERSIONING_SCHEME}}', description: 'How sous determines build / deploy version', name: 'SOUS_VERSIONING_SCHEME')
      string(defaultValue: '{{SOUS_STATIC_TEST}}', description: 'Execute Static Tests', name: 'SOUS_STATIC_TEST')
      string(defaultValue: '{{SOUS_STATIC_TEST_COMMAND}}', description: 'Static Tests Command', name: 'SOUS_STATIC_TEST_COMMAND')
      string(defaultValue: '{{SOUS_UNIT_TEST}}', description: 'Execute Unit Tests', name: 'SOUS_UNIT_TEST')
      string(defaultValue: '{{SOUS_UNIT_TEST_COMMAND}}', description: 'Unit Tests Command', name: 'SOUS_UNIT_TEST_COMMAND')
      string(defaultValue: '{{SOUS_SMOKE_TEST}}', description: 'Execute Smoke Tests', name: 'SOUS_SMOKE_TEST')
      string(defaultValue: '{{SOUS_SMOKE_TEST_COMMAND}}', description: 'Smoke Tests Command', name: 'SOUS_SMOKE_TEST_COMMAND')
      string(defaultValue: '{{SOUS_INTEGRATION_TEST}}', description: 'Execute Integration Tests', name: 'SOUS_INTEGRATION_TEST')
      string(defaultValue: '{{SOUS_INTEGRATION_TEST_COMMAND}}', description: 'Integration Tests Command', name: 'SOUS_INTEGRATION_TEST_COMMAND')
      string(defaultValue: '{{SOUS_DEPLOY_CI}}', description: 'Deploy to CI', name: 'SOUS_DEPLOY_CI')
      string(defaultValue: '{{SOUS_POST_CI_TEST}}', description: 'Test to run after CI', name: 'SOUS_POST_CI_TEST')
      string(defaultValue: '{{SOUS_POST_CI_TEST_COMMAND}}', description: 'Test Command to run after CI', name: 'SOUS_POST_CI_TEST_COMMAND')
      string(defaultValue: '{{SOUS_DEPLOY_PP}}', description: 'Deploy to PP', name: 'SOUS_DEPLOY_PP')
      string(defaultValue: '{{SOUS_POST_PP_TEST}}', description: 'Test to run after PP', name: 'SOUS_POST_PP_TEST')
      string(defaultValue: '{{SOUS_POST_PP_TEST_COMMAND}}', description: 'Test Command to run after PP', name: 'SOUS_POST_PP_TEST_COMMAND')
      //Note we can make this a pop-up if people want to be be gated and asked if deploy to prod (manual check before prod push)
      string(defaultValue: '{{SOUS_DEPLOY_PROD}}', description: 'Deploy to PROD', name: 'SOUS_DEPLOY_PROD')
      string(defaultValue: '{{SOUS_POST_PROD_TEST}}', description: 'Test to run after Prod', name: 'SOUS_POST_PROD_TEST')
      string(defaultValue: '{{SOUS_POST_PROD_TEST_COMMAND}}', description: 'Test Command to run after Prod', name: 'SOUS_POST_PROD_TEST_COMMAND')
      //Could introduce the negative, if SOUS_USE_RC == 'YES', then don't deploy to other environments
      string(defaultValue: '{{SOUS_USE_RC}}', description: 'Deploy to RC', name: 'SOUS_USE_RC')
	  //If yes, the deploy step will wait a day asking if should continue deploying to PROD
      string(defaultValue: '{{SOUS_DEPLOY_PROD_QUERY_USER}}', description: 'Ask user before deploying to PROD', name: 'SOUS_DEPLOY_PROD_QUERY_USER')
  }
  triggers {
    pollSCM('H/5 * * * *')
  }
  stages {
    //Immediately send github PR all checks that this pipeline will be checking
    stage('Git statuses tests') {
      parallel {
        stage('Static') {
          when{
            expression { params.SOUS_STATIC_TEST == 'YES' }
          }
          steps {
            githubNotify context: 'Jenkins/Test/Static-Check', description: 'Static Check Tests', status: 'PENDING'
          }
        }
        stage('Unit') {
          when{
            expression { params.SOUS_UNIT_TEST == 'YES' }
          }
          steps {
            githubNotify context: 'Jenkins/Test/Unit', description: 'Unit Tests', status: 'PENDING'
          }
        }
        stage('Smoke') {
          when{
            expression { params.SOUS_SMOKE_TEST == 'YES' }
          }
          steps {
            githubNotify context: 'Jenkins/Test/Smoke', description: 'Smoke Tests', status: 'PENDING'
          }
        }
        stage('Integration') {
          when{
            expression { params.SOUS_INTEGRATION_TEST == 'YES' }
          }
          steps {
            githubNotify context: 'Jenkins/Test/Integration', description: 'Integration Tests', status: 'PENDING'
          }
        }
      }
    }
    stage('Git statuses build') {
      steps {
        githubNotify context: 'Jenkins/Build', description: 'Build', status: 'SUCCESS'
          githubNotify context: 'Jenkins Overall Success', description: 'Pipeline Status', status: 'PENDING'
      }
    }
    stage('Git statuses deploy CI') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_DEPLOY_CI == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Deploy/CI-SF', description: 'Deploy to CI-SF', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/CI-USWEST2', description: 'Deploy to CI-USWEST2', status: 'PENDING'

      }
    }
    stage('Git statuses Post CI Test') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_POST_CI_TEST == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Test/Post/CI', description: 'Post Test of CI', status: 'PENDING'
      }
    }
    stage('Git statuses deploy PP') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_DEPLOY_PP == 'YES' }
      }
      steps {
          githubNotify context: 'Jenkins/Deploy/PP-SF', description: 'Deploy to PP-SF', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/PP-USWEST2', description: 'Deploy to PP-USWEST2', status: 'PENDING'

      }
    }
    stage('Git statuses Post PP Test') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_POST_PP_TEST == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Test/Post/PP', description: 'Post Test of PP', status: 'PENDING'
      }
    }
    stage('Git statuses PROD') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_DEPLOY_PROD == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Deploy/PROD-USWEST2', description: 'Deploy to PROD-USWEST2', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/PROD-EUWEST1', description: 'Deploy to PROD-EUWEST1', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/PROD-LN', description: 'Deploy to PROD-LN', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/PROD-SC', description: 'Deploy to PROD-SC', status: 'PENDING'
      }
    }
    stage('Git statuses Post Prod Test') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_POST_PROD_TEST == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Test/Post/PROD', description: 'Post Test of Prod', status: 'PENDING'
      }
    }
    stage('Git statuses RC') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
          expression { params.SOUS_USE_RC == 'YES' }
      }
      steps {
        githubNotify context: 'Jenkins/Deploy/RCCI-SF', description: 'Deploy to RCCI-SF', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/RCPP-SF', description: 'Deploy to RCPP-SF', status: 'PENDING'
          githubNotify context: 'Jenkins/Deploy/RCPROD-SC', description: 'Deploy to RCPROD-SC', status: 'PENDING'
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
          when{
            expression { params.SOUS_STATIC_TEST == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_STATIC_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
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
          when{
            expression { params.SOUS_UNIT_TEST == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_UNIT_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
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
        stage('Smoke') {
          when{
            expression { params.SOUS_SMOKE_TEST == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_SMOKE_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Smoke', description: 'Smoke Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Smoke', description: 'Smoke Tests Failed', status: 'FAILURE'
            }
          }
        }
        stage('Integration') {
          when{
            expression { params.SOUS_INTEGRATION_TEST == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_INTEGRATION_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Integration', description: 'Integration Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Integration', description: 'Integration Tests Failed', status: 'FAILURE'
            }
          }
        }
      }
    }
    stage('Determine SOUS_TAG') {
      agent { label 'mesos-qa-uswest2' }
      steps {
        script {
          def tag = new com.opentable.sous.Tag()
            tag.execute()
        }
        echo "SOUS_TAG = ${env.SOUS_TAG}"
      }
    }
    stage('Determine SOUS_USER') {
      agent { label 'mesos-qa-uswest2' }
      steps {
        script {
          def tag = new com.opentable.sous.User()
            tag.execute()
        }
        echo "SOUS_USER = ${env.SOUS_USER}"
      }
    }
    stage('Determine SOUS_EMAIL') {
      agent { label 'mesos-qa-uswest2' }
      steps {
        script {
          def tag = new com.opentable.sous.Email()
            tag.execute()
        }
        echo "SOUS_EMAIL= ${env.SOUS_EMAIL}"
      }
    }
    stage('Build') {
      options {
        timeout(time: 30, unit: 'MINUTES')
      }
      steps {
        echo 'Build in Jenkinsfile'
          retry(2) {
            script {
              def build = new com.opentable.sous.Build()
                build.execute()
            }
          }
        echo 'leaving Jenkinsfile stage build'
      }
      environment {
        SOUS_CMD_TAG = "latest"
      }
      post {
        success {
          githubNotify context: 'Jenkins/Build', description: 'Build', status: 'SUCCESS'
        }
        failure {
          githubNotify context: 'Jenkins/Build', description: 'Build', status: 'FAILURE'
        }
      }
    }
    stage('{{SOUS_RELEASE_BRANCH}} branch deploy CI'){
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
        expression { params.SOUS_DEPLOY_CI == 'YES' }
      }
      parallel {
        stage('Deploy ci-sf') {
          when{
            expression { params.SOUS_DEPLOY_CI == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "ci-sf"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/CI-SF', description: 'Deploy to CI-SF', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/CI_SF', description: 'Deploy to CI-SF', status: 'FAILURE'
            }
          }
        }
        stage('Deploy ci-uswest2') {
          when{
            expression { params.SOUS_DEPLOY_CI == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "ci-uswest2"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/CI-USWEST2', description: 'Deploy to CI-USWEST2', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/CI-USWEST2', description: 'Deploy to CI-USWEST2', status: 'FAILURE'
            }
          }
        }
        stage('Deploy rcci-sf') {
          when{
            expression { params.SOUS_DEPLOY_CI == 'YES' }
            expression { params.SOUS_USE_RC == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "rcci-sf"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/RCCI-SF', description: 'Deploy to RCCI-SF', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/RCCI-SF', description: 'Deploy to RCCI-SF', status: 'FAILURE'
            }
          }
        }
	  }
	}
        stage('Test Post CI') {
          when{
            expression { params.SOUS_POST_CI_TEST == 'YES' }
			branch '{{SOUS_RELEASE_BRANCH}}'
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_POST_CI_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Post/CI', description: 'Post CI Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Post/CI', description: 'Post CI Tests Failed', status: 'FAILURE'
            }
          }
        }
    stage('{{SOUS_RELEASE_BRANCH}} branch deploy PP'){
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
        expression { params.SOUS_DEPLOY_PP == 'YES' }
      }
      parallel {
        stage('Deploy pp-sf') {
          when{
            expression { params.SOUS_DEPLOY_PP == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "pp-sf"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PP-SF', description: 'Deploy to PP-SF', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PP-SF', description: 'Deploy to PP-SF', status: 'FAILURE'
            }
          }
        }
        stage('Deploy pp-uswest2') {
          when{
            expression { params.SOUS_DEPLOY_PP == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "pp-uswest2"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PP-USWEST2', description: 'Deploy to PP-USWEST2', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PP-USWEST2', description: 'Deploy to PP-USWEST2', status: 'FAILURE'
            }
          }
        }
        stage('Deploy rcpp-sf') {
          when{
            expression { params.SOUS_DEPLOY_PP == 'YES' }
            expression { params.SOUS_USE_RC == 'YES' }
          }
          agent { label 'mesos-qa-uswest2' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "rcpp-sf"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/RCPP-SF', description: 'Deploy to RCPP-SF', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/RCPP-SF', description: 'Deploy to RCPP-SF', status: 'FAILURE'
            }
          }
        }
	  }
	}
        stage('Test Post PP') {
          when{
            expression { params.SOUS_POST_PP_TEST == 'YES' }
			branch '{{SOUS_RELEASE_BRANCH}}'
          }
          agent { label 'mesos-qa-uswest2' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_POST_PP_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Post/PP', description: 'Post PP Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Post/PP', description: 'Post PP Tests Failed', status: 'FAILURE'
            }
          }
        }
	  stage('Input Prod Deployment') {
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
        expression { params.SOUS_DEPLOY_PROD_QUERY_USER == 'YES' }
      }
      steps {
        script {
        	env.DEPLOY_TO_PROD = input message: 'User input required',
          parameters: [choice(name: 'Deploy to Prod?', choices: 'no\nyes', description: 'Choose "yes" if you want to deploy this build to production.')]
        }
      }
		}
    stage('{{SOUS_RELEASE_BRANCH}} branch deploy PROD'){
      when{
        branch '{{SOUS_RELEASE_BRANCH}}'
        expression { params.SOUS_DEPLOY_PROD == 'YES' }
      }
      parallel {
		stage('Deploy rcprod-sc') {
          when{
            expression { params.SOUS_DEPLOY_PROD == 'YES' }
            expression { params.SOUS_USE_RC == 'YES' }
          }
          agent { label 'mesos-prod-sc' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "rcprod-sc"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/RCPROD-SC', description: 'Deploy to RCPROD-SC', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/RCPROD-SC', description: 'Deploy to RCPROD-SC', status: 'FAILURE'
            }
          }
        }
        stage('Deploy prod-sc') {
          when{
            expression { params.SOUS_DEPLOY_PROD == 'YES' }
          }
          agent { label 'mesos-prod-sc' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "prod-sc"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PROD-SC', description: 'Deploy to PROD-SC', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PROD-SC', description: 'Deploy to PROD-SC', status: 'FAILURE'
            }
          }
        }
        stage('Deploy prod-ln') {
          when{
            expression { params.SOUS_DEPLOY_PROD == 'YES' }
          }
          agent { label 'mesos-prod-sc' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "prod-ln"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PROD-LN', description: 'Deploy to PROD-LN', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PROD-LN', description: 'Deploy to PROD-LN', status: 'FAILURE'
            }
          }
        }
        stage('Deploy prod-euwest1') {
          when{
            expression { params.SOUS_DEPLOY_PROD == 'YES' }
          }
          agent { label 'mesos-prod-sc' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "prod-euwest1"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PROD-EUWEST1', description: 'Deploy to PROD-EUWEST1', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PROD-EUWEST1', description: 'Deploy to PROD-EUWEST1', status: 'FAILURE'
            }
          }
        }
        stage('Deploy prod-uswest2') {
          when{
            expression { params.SOUS_DEPLOY_PROD == 'YES' }
          }
          agent { label 'mesos-prod-sc' }
          options {
            timeout(time: 5, unit: 'MINUTES')
          }
          steps {
            retry(4) {
              script {
                def deploy = new com.opentable.sous.Deploy()
                  deploy.execute()
              }
            }
          }
          environment {
            SOUS_CLUSTER = "prod-uswest2"
          }
          post {
            success {
              githubNotify context: 'Jenkins/Deploy/PROD-USWEST2', description: 'Deploy to PROD-USWEST2', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Deploy/PROD-USWEST2', description: 'Deploy to PROD-USWEST2', status: 'FAILURE'
            }
          }
        }
	  }
	}
        stage('Test Post Prod') {
          when{
            expression { params.SOUS_POST_PROD_TEST == 'YES' }
			branch '{{SOUS_RELEASE_BRANCH}}'
          }
          agent { label 'mesos-prod-sc' }
          steps {
            withEnv(["CMD_TO_EXECUTE=${params.SOUS_POST_PROD_TEST_COMMAND}"]) {
              script {
                def executecmd = new com.opentable.sous.ExecuteCmd()
                executecmd.execute()
              }
            }
          }
          post {
            success {
              githubNotify context: 'Jenkins/Test/Post/PROD', description: 'Post Prod Tests Passed', status: 'SUCCESS'
            }
            failure {
              githubNotify context: 'Jenkins/Test/Post/PROD', description: 'Post Prod Tests Failed', status: 'FAILURE'
            }
          }
        }
  }
  post {
    always {
      echo 'This will always run'

        script {
          def notifier = new org.gradiant.jenkins.slack.SlackNotifier()

            env.SLACK_CHANNEL = '#team-eng-sous-bots, #tech-deploy'
            env.CHANGE_LIST = 'true'
            env.NOTIFY_SUCCESS = 'true'

            notifier.notifyResult()
        }

      //slackSend color: 'good', message: 'Message from Jenkins Pipeline'
      //script {
      //  def slack = new com.opentable.sous.Slack()
      //  slack.call(currentBuild.currentResult, '#team-eng-sous-bots')
      //}
    }
    success {
      echo 'This will run only if successful'
        githubNotify context: 'Jenkins Overall Success', description: 'PIPELINE all Passed!!!', status: 'SUCCESS'
    }
    failure {
      echo 'This will run only if failed'
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
`

	return template
}
