import { useAppContext } from 'providers/App/App';
import APIError from 'types/api/error';
import { LicensePlatform } from 'types/api/licensesV3/getActive';

export const useGetTenantLicense = (): {
	isCloudUser: boolean;
	isEnterpriseSelfHostedUser: boolean;
	isCommunityUser: boolean;
	isCommunityEnterpriseUser: boolean;
} => {
	const { activeLicense, activeLicenseFetchError } = useAppContext();

	const responsePayload = {
		isCloudUser: activeLicense?.platform === LicensePlatform.CLOUD || false,
		isEnterpriseSelfHostedUser:
			activeLicense?.platform === LicensePlatform.SELF_HOSTED || false,
		isCommunityUser: false,
		isCommunityEnterpriseUser: false,
	};

	// 정규화를 거치지 않은 에러(raw TypeError 등)가 컨텍스트에 들어와도
	// getHttpStatusCode 호출로 2차 크래시하지 않도록 APIError만 신뢰한다.
	const licenseErrorStatus =
		activeLicenseFetchError instanceof APIError
			? activeLicenseFetchError.getHttpStatusCode()
			: undefined;

	if (licenseErrorStatus === 404) {
		responsePayload.isCommunityEnterpriseUser = true;
	}

	if (licenseErrorStatus === 501) {
		responsePayload.isCommunityUser = true;
	}

	return responsePayload;
};
