'use strict';

var app = angular.module('app', []);

app.controller('AssocCtrl', ['$scope', '$http', '$location', function($scope, $http, $location) {
  $scope.domainForm = {
    newDomain: ''
  };
  $scope.associations = [];
  $scope.getAssocs = function() {
    $http.get('/app/association').
      success(function(data) {
        $scope.associations = data.associations;
      }); // TODO error handling
  };
  $scope.addAssoc = function() {
    var domain = {
      domain: $scope.domainForm.newDomain
    };
    $scope.domainForm.error = null;
    $http.post('/app/association', domain).
      success(function(data, status) {
        $scope.domainForm.newDomain = '';
        $scope.associations.push(domain);
      }).
      error(function(data, status) {
        $scope.domainForm.error = data;
      });
  };
  $scope.signOut = function() {
    $scope.associations = [];
    $scope.domainForm.newDomain = '';
    $http.delete('/app/session').
      success(function() { window.location.replace('/app/'); }).
      error(function() { window.location.replace('/index.html'); });
  };

  $scope.getAssocs();
}]);
