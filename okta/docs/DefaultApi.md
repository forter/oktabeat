# \DefaultApi

All URIs are relative to *https://example.okta.com/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetLogs**](DefaultApi.md#GetLogs) | **Get** /logs | Get logs


# **GetLogs**
> map[string]interface{} GetLogs(ctx, accept, contentType, authorization, optional)
Get logs

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **accept** | **string**|  | [default to application/json]
  **contentType** | **string**|  | [default to application/json]
  **authorization** | **string**|  | 
 **optional** | ***GetLogsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetLogsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **since** | **optional.Time**|  | 
 **until** | **optional.Time**|  | 
 **after** | **optional.String**|  | 
 **filter** | **optional.String**|  | 
 **q** | **optional.String**|  | 
 **sortOrder** | **optional.String**|  | [default to ASCENDING]
 **limit** | **optional.Int32**|  | [default to 1000]

### Return type

[**map[string]interface{}**](map[string]interface{}.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

