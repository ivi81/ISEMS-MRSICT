package routingflowsmoduleapirequestprocessing

import (
	"encoding/json"
	"fmt"
	"regexp"

	"ISEMS-MRSICT/commonlibs"
	"ISEMS-MRSICT/datamodels"
	"ISEMS-MRSICT/decoders"

	"github.com/asaskevich/govalidator"
)

//UnmarshalJSONObjectSTIXReq декодирует JSON документ, поступающий от модуля 'moduleapirequestprocessing', который содержит список объектов STIX
func UnmarshalJSONObjectSTIXReq(msgReq datamodels.ModAPIRequestProcessingReqJSON) ([]*datamodels.ElementSTIXObject, error) {
	var listSTIXObjectJSON []*json.RawMessage

	if err := json.Unmarshal(*msgReq.RequestDetails, &listSTIXObjectJSON); err != nil {
		return nil, err
	}

	listResults, err := decoders.GetListSTIXObjectFromJSON(listSTIXObjectJSON)
	if err != nil {
		return nil, err
	}

	return listResults, nil
}

//UnmarshalJSONObjectReqSearchParameters декодирует JSON документ, поступающий от модуля 'moduleapirequestprocessing', который содержит параметры для
// выполнения поиска документов в коллекциях БД
func UnmarshalJSONObjectReqSearchParameters(msgReq *json.RawMessage) (datamodels.ModAPIRequestProcessingResJSONSearchReqType, error) {
	var result datamodels.ModAPIRequestProcessingResJSONSearchReqType
	var resultTmp datamodels.CommonModAPIRequestProcessingResJSONSearchReqType

	if err := json.Unmarshal(*msgReq, &resultTmp); err != nil {
		return result, err
	}

	switch resultTmp.CollectionName {
	case "stix object":
		var msgSearch datamodels.SearchThroughCollectionSTIXObjectsType
		if err := json.Unmarshal(*resultTmp.SearchParameters, &msgSearch); err != nil {
			return result, err
		}

		result.SearchParameters = msgSearch

	case "":

	}

	return result, nil
}

//CheckSearchSTIXObject выполняет валидацию параметров запроса, к поисковой машине, который выполняется для поиска информации по STIX объектам
func CheckSearchSTIXObject(req *datamodels.ModAPIRequestProcessingResJSONSearchReqType) (datamodels.ModAPIRequestProcessingResJSONSearchReqType, error) {
	sp, ok := req.SearchParameters.(datamodels.SearchThroughCollectionSTIXObjectsType)
	if !ok {
		return *req, fmt.Errorf("type conversion error")
	}

	if len(sp.DocumentsID) > 0 {
		for _, v := range sp.DocumentsID {
			if !(regexp.MustCompile(`^[a-z](--)[0-9a-f|-]+$`).MatchString(v)) {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}
	}

	if len(sp.DocumentsType) > 0 {
		for _, v := range sp.DocumentsType {
			if !(regexp.MustCompile(`^[0-9a-z|-]+$`).MatchString(v)) {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}
	}

	tcsn := sp.Created.Start.Unix()
	tcen := sp.Created.End.Unix()

	if tcsn > 0 && tcen > 0 {
		if tcsn >= tcen {
			return *req, fmt.Errorf("invalid search value accepted")
		}
	}

	tmsn := sp.Modified.Start.Unix()
	tmen := sp.Modified.End.Unix()

	if tmsn > 0 && tmen > 0 {
		if tmsn >= tmen {
			return *req, fmt.Errorf("invalid search value accepted")
		}
	}

	sp.CreatedByRef = commonlibs.StringSanitize(sp.CreatedByRef)

	if len(sp.SpecificSearchFields) == 0 {
		return *req, nil
	}

	for k, v := range sp.SpecificSearchFields {
		sp.SpecificSearchFields[k].SearchFields.Name = commonlibs.StringSanitize(v.SearchFields.Name)

		if len(v.SearchFields.Aliases) > 0 {
			for key, value := range v.SearchFields.Aliases {
				sp.SpecificSearchFields[k].SearchFields.Aliases[key] = commonlibs.StringSanitize(value)
			}
		}

		tcsn := v.SearchFields.FirstSeen.Start.Unix()
		tcen := v.SearchFields.FirstSeen.End.Unix()

		if tcsn > 0 && tcen > 0 {
			if tcsn >= tcen {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}

		if len(v.SearchFields.Roles) > 0 {
			for key, value := range v.SearchFields.Roles {
				sp.SpecificSearchFields[k].SearchFields.Roles[key] = commonlibs.StringSanitize(value)
			}
		}

		if v.SearchFields.Country != "" {
			if !(regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(v.SearchFields.Country)) {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}

		sp.SpecificSearchFields[k].SearchFields.City = commonlibs.StringSanitize(v.SearchFields.City)

		if v.SearchFields.URL != "" {
			if !govalidator.IsURL(v.SearchFields.URL) {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}

		if len(v.SearchFields.Value) > 0 {
			if ok := checkSearchFieldsValue(req.CollectionName, v.SearchFields.Value); !ok {
				return *req, fmt.Errorf("invalid search value accepted")
			}
		}
	}

	return *req, nil
}

func checkSearchFieldsValue(valueType string, l []string) bool {
	for _, v := range l {
		switch valueType {
		case "domain-name":
			if !govalidator.IsDNSName(v) {
				return false
			}

		case "email-addr":
			if !govalidator.IsEmail(v) {
				return false
			}

		case "ipv4-addr":
			isIPv4 := commonlibs.IsIPv4Address(v)
			isNetworkIPv4 := commonlibs.IsComputerNetAddrIPv4Range(v)
			if !isIPv4 && !isNetworkIPv4 {
				return false
			}

		case "ipv6-addr":
			/*
						!!!!!!!!!!!!!!!!!!
				Здесь все же надо сделать проверку IPv6

			*/

		case "url":
			if !govalidator.IsURL(v) {
				return false
			}
		}
	}

	return true
}

//CheckSTIXObjects выполняет валидацию списка STIX объектов
func CheckSTIXObjects(l []*datamodels.ElementSTIXObject) error {
	for _, item := range l {
		if item.Data.CheckingTypeFields() {
			continue
		}

		fmt.Printf("Error checking type STIX object: '%s'\n", item.DataType)

		return fmt.Errorf("one or more STIX objects are invalid")
	}

	return nil
}

//SanitizeSTIXObject выполняем санитаризацию полученных STIX объектов
func SanitizeSTIXObject(l []*datamodels.ElementSTIXObject) []*datamodels.ElementSTIXObject {
	var elem datamodels.HandlerSTIXObject
	listElements := make([]*datamodels.ElementSTIXObject, 0, len(l))

	for _, item := range l {
		switch element := item.Data.(type) {
		case datamodels.AttackPatternDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.CampaignDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.CourseOfActionDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.GroupingDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.IdentityDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.IndicatorDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.InfrastructureDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.IntrusionSetDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.LocationDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.MalwareDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.MalwareAnalysisDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.NoteDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ObservedDataDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.OpinionDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ReportDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ThreatActorDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ToolDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.VulnerabilityDomainObjectsSTIX:
			elem = element.SanitizeStruct()

		case datamodels.RelationshipObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.SightingObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ArtifactCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.AutonomousSystemCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.DirectoryCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.DomainNameCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.EmailAddressCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.EmailMessageCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.FileCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.IPv4AddressCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.IPv6AddressCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.MACAddressCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.MutexCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.NetworkTrafficCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.ProcessCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.SoftwareCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.URLCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.UserAccountCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.WindowsRegistryKeyCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		case datamodels.X509CertificateCyberObservableObjectSTIX:
			elem = element.SanitizeStruct()

		}

		if elem == nil {
			continue
		}

		listElements = append(listElements, &datamodels.ElementSTIXObject{
			DataType: item.DataType,
			Data:     elem,
		})
	}

	return listElements
}
