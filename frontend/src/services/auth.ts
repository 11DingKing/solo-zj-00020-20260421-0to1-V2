import api from "./api";

export interface LoginResponse {
  token: string;
  user: {
    id: number;
    username: string;
    email?: string;
  };
}

export const login = (
  username: string,
  password: string,
): Promise<LoginResponse> => {
  return api
    .post("/auth/login", {
      username,
      password,
    })
    .then((res) => res.data);
};

export const register = (
  username: string,
  password: string,
  email?: string,
) => {
  return api
    .post("/auth/register", {
      username,
      password,
      email,
    })
    .then((res) => res.data);
};

export const logout = () => {
  localStorage.removeItem("token");
};

export const getCurrentUser = () => {
  return api.get("/auth/me").then((res) => res.data);
};
