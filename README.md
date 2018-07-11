# gppStandby

testing the following scenarios
```
getRolloverdate()
getWAITSCHEDSUBBATCHcount()
edoTrackingFileSAPLEG()
getMPWAITcount()
getSCHEDULEcount()
edoFilesOutGoing()
checkFailureFolders()
edoFilesOutGoingArchived()
edoResponseLEG()
edoResponseSAP() 
```
 
```

kubectl --context=legion-sdc -n gppstandby-sit create configmap gppstandby-config-map --from-env-file=env.list

```


